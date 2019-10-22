package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/spf13/cobra"
)

type filelistFile struct {
	filename string
	file     *os.File
}

func (f filelistFile) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("filename", f.filename)
	return nil
}

type filelistFilter func(f filelistFile) (bool, error)

func filterNone() filelistFilter {
	return func(f filelistFile) (bool, error) {
		return true, nil
	}
}

func filterModify(time time.Time, after bool) filelistFilter {
	return func(f filelistFile) (bool, error) {
		fileInfo, err := f.file.Stat()
		if err != nil {
			return false, err
		}
		modTime := fileInfo.ModTime()
		if after {
			return modTime.After(time), nil
		}
		return modTime.Before(time), nil
	}
}

type filelistAction func(f filelistFile) error

func composeFilelistActions(actA filelistAction, actB filelistAction) filelistAction {
	return func(f filelistFile) error {
		if err := actA(f); err != nil {
			return err
		}
		if err := actB(f); err != nil {
			return err
		}
		return nil
	}
}

func actLog() filelistAction {
	return func(f filelistFile) error {
		logger.Info("taking action on file",
			zap.Object("filelistFile", f),
		)
		return nil
	}
}

func actList(out io.StringWriter) filelistAction {
	return func(f filelistFile) error {
		_, err := out.WriteString(f.filename + "\n")
		return err
	}
}

func actUnlink() filelistAction {
	return func(f filelistFile) error {
		return os.Remove(f.filename)
	}
}

type filelistActionRes struct {
	err          error
	filtered     bool
	filelistFile filelistFile
}

func (res filelistActionRes) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if res.err != nil {
		enc.AddString("err", res.err.Error())
	}
	enc.AddBool("filtered", res.filtered)
	if err := enc.AddObject("filelistFile", res.filelistFile); err != nil {
		return err
	}
	return nil
}

type filelistActionChansT struct {
	filename     chan string
	allEnts      chan filelistFile
	filteredEnts chan filelistFile
	res          chan filelistActionRes
}

func buildFilelistEntrySync(filename string) (filelistFile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return filelistFile{
			filename: filename,
		}, err
	}
	return filelistFile{
		filename: filename,
		file:     file,
	}, nil
}

func buildFilelistEntries(filelistActionChans filelistActionChansT) {
	var filename string
	for {
		filename = <-filelistActionChans.filename
		ent, err := buildFilelistEntrySync(filename)
		if err != nil {
			filelistActionChans.res <- filelistActionRes{
				err:          err,
				filelistFile: ent,
			}
			continue
		}
		filelistActionChans.allEnts <- ent
	}
}

func filterFilelistEntries(filelistActionChans filelistActionChansT, filter filelistFilter) {
	var ent filelistFile
	for {
		ent = <-filelistActionChans.allEnts
		ok, err := filter(ent)
		if err != nil {
			filelistActionChans.res <- filelistActionRes{
				err:          err,
				filelistFile: ent,
			}
			continue
		}
		if !ok {
			filelistActionChans.res <- filelistActionRes{
				filtered:     true,
				filelistFile: ent,
			}
			continue
		}
		filelistActionChans.filteredEnts <- ent
	}
}

func actOnFilelistEntries(filelistActionChans filelistActionChansT, action filelistAction) {
	var ent filelistFile
	for {
		ent = <-filelistActionChans.filteredEnts
		err := action(ent)
		if err != nil {
			filelistActionChans.res <- filelistActionRes{
				err:          err,
				filelistFile: ent,
			}
			continue
		}
		filelistActionChans.res <- filelistActionRes{
			filelistFile: ent,
		}
	}
}

const (
	parallelismPerBuildFilelistWorkers  = 20
	parallelismPerFilterFilelistWorkers = 20
	parallelismPerActOnWorkers          = 20
)

func actOnFilelist(inputFile *os.File, action filelistAction, filter filelistFilter) error {

	var wgIndiv sync.WaitGroup
	var wgAllInit sync.WaitGroup

	scanner := bufio.NewScanner(inputFile)

	filelistActionChans := filelistActionChansT{
		filename:     make(chan string),
		allEnts:      make(chan filelistFile),
		filteredEnts: make(chan filelistFile),
		res:          make(chan filelistActionRes),
	}

	buildFilelistWorkers := unlinkParams.parallelism / parallelismPerBuildFilelistWorkers
	filterFilelistWorkers := unlinkParams.parallelism / parallelismPerFilterFilelistWorkers
	actOnWorkers := unlinkParams.parallelism / parallelismPerActOnWorkers

	for i := 0; i < buildFilelistWorkers; i++ {
		go buildFilelistEntries(filelistActionChans)
	}
	for i := 0; i < filterFilelistWorkers; i++ {
		go filterFilelistEntries(filelistActionChans, filter)
	}
	for i := 0; i < actOnWorkers; i++ {
		go actOnFilelistEntries(filelistActionChans, action)
	}

	logger.Info("started worker threads",
		zap.Int("buildFilelist", buildFilelistWorkers),
		zap.Int("filterFilelist", filterFilelistWorkers),
		zap.Int("actOn", actOnWorkers),
	)

	wgAllInit.Add(1)
	go func() {
		for scanner.Scan() {
			wgIndiv.Add(1)
			filelistActionChans.filename <- scanner.Text()
		}
		wgAllInit.Done()
	}()

	var err error
	var errCnt int
	var filteredCnt int
	var actionCnt int

	go func() {
		var res filelistActionRes
		for {
			res = <-filelistActionChans.res
			logger.Info("got result",
				zap.Object("res", res),
			)
			switch {
			case res.err != nil:
				err = res.err
				errCnt++
			case res.filtered:
				filteredCnt++
			default:
				actionCnt++
			}
			wgIndiv.Done()
		}
	}()

	wgAllInit.Wait()
	wgIndiv.Wait()

	return err
}

func parseTime(timeStr string) (time.Time, error) {
	var t time.Time
	var err error

	// parse based on reference time
	// Mon Jan 2 15:04:05 -0700 MST 2006
	validFormats := []string{
		"2006-Jan-02",
		"0601021504",
		"060102150405",
	}

	for _, format := range validFormats {
		t, err = time.Parse(format, timeStr)
		if err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(),
				t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
				time.Local), nil
		}
	}
	return time.Time{}, fmt.Errorf("time string didn't match any valid format")
}

var actOnFilelistCmd = &cobra.Command{
	Use:   "filelist-actions",
	Short: "perform various operations on a list of files",
	Long:  "perform various operations on a list of files",
	Run: func(cmd *cobra.Command, args []string) {
		var action filelistAction
		var filter filelistFilter
		var inputFile *os.File

		action = actLog()

		if unlinkParams.unlink {
			action = composeFilelistActions(action, actUnlink())
		}

		if unlinkParams.out != "" {
			var outputFile *os.File
			if unlinkParams.out == "-" {
				outputFile = os.Stdout
			} else {
				var err error
				outputFile, err = os.Create(unlinkParams.out)
				if err != nil {
					log.Fatalf("failed to open output file '%v': '%v'", unlinkParams.out, err)
				}
			}
			action = composeFilelistActions(action, actList(outputFile))
		}

		if unlinkParams.timeBefore == "" {
			filter = filterNone()
		} else {
			t, err := parseTime(unlinkParams.timeBefore)
			if err != nil {
				log.Fatalf("failed to parse --time-before: %v", err)
			}
			filter = filterModify(t, false)
		}

		if unlinkParams.filelist == "" || unlinkParams.filelist == "-" {
			inputFile = os.Stdin
		} else {
			var err error
			inputFile, err = os.Open(unlinkParams.filelist)
			if err != nil {
				log.Fatalf("failed to open input file '%v': '%v'", unlinkParams.filelist, err)
			}
		}

		if err := actOnFilelist(inputFile, action, filter); err != nil {
			log.Fatalf("failed to act on filelist: '%v'", err)
		}
	},
}

var unlinkParams struct {
	filelist    string
	out         string
	timeBefore  string
	unlink      bool
	parallelism int
}

func init() {
	flags := actOnFilelistCmd.Flags()
	flags.StringVarP(&unlinkParams.filelist, "filelist", "", "", "List of input files")
	flags.StringVarP(&unlinkParams.out, "out", "", "", "Output list of files")
	flags.StringVarP(&unlinkParams.timeBefore, "time-before", "", "", "Filter modify times before.")
	flags.BoolVarP(&unlinkParams.unlink, "unlink", "", false, "Whether to unlink files.")
	flags.IntVarP(&unlinkParams.parallelism, "parallelism", "", 100, "Amount of parallelism.")

	rootCmd.AddCommand(actOnFilelistCmd)
}
