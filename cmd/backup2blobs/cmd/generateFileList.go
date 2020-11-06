package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/oneconcern/datamon/pkg/convert"

	"go.uber.org/zap"

	iradix "github.com/hashicorp/go-immutable-radix"

	"github.com/karrick/godirwalk"
	"github.com/spf13/cobra"
)

type task struct {
	directories    *iradix.Tree // List of directories to be processed
	readDirOpCount int          // the number of readdirs to issue
	childTasks     int          // The number of child tasks spawned
	signal         sync.Mutex   // Lock it to wait for it to be done.
	parentTask     *task        // Originating task
	lock           sync.Mutex   // Protect the task
}

func newTask(directories *iradix.Tree, parent *task) *task {
	d := task{
		directories: directories,
		signal:      sync.Mutex{},
		parentTask:  parent,
	}
	d.signal.Lock() // Unlocked when done.
	return &d
}

func (d *task) dirCount() int {
	return d.directories.Len()
}

func (d *task) iterator() *iradix.Iterator {
	return d.directories.Root().Iterator()
}

func (d *task) incChildTasks(count int) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.childTasks += count
}

func (d *task) childDone() {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.childTasks--
	// Process further only if read dirs have been issued.
	// Multiple tasks can be issued for a parent task and
	// further processing should be done only if read dirs
	// have been issued.
	if d.readDirOpCount == d.dirCount() {
		if d.childTasks == 0 {
			if d.parentTask != nil {
				d.parentTask.childDone()
			}
			d.signal.Unlock()
		}
	}
}

func (d *task) incReadDirOpCount() {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.readDirOpCount++
}

func (d *task) done() {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.readDirOpCount == d.dirCount() {
		// All directories processed.
		if d.childTasks == 0 {
			// No child tasks spawned. This task is done, inform parent.
			if d.parentTask != nil {
				d.parentTask.childDone()
			}
			d.signal.Unlock()
		}
	}
}

func walkDirConcurrently(rootDir string) {
	dirChannel := make(chan *task, generateParams.dirChannelSize)
	fileChannel := make(chan string, generateParams.fileChannelSize)

	var wg sync.WaitGroup
	wg.Add(1)
	go processFiles(fileChannel, &wg)

	r := iradix.New()
	r, _, update := r.Insert(convert.UnsafeStringToBytes(filepath.Clean(rootDir)), nil)
	if update {
		logger.Error("Failed to insert root")
		return
	}
	dirTask := newTask(r, nil)

	dirChannel <- dirTask

	for i := 0; i < generateParams.dirRoutines; i++ {
		go processDir(dirChannel, fileChannel)
	}
	dirTask.signal.Lock()
	close(dirChannel)
	close(fileChannel)
	wg.Wait()
}

func generateParamsToOutputFile() (*os.File, error) {
	var file *os.File
	var err error
	if generateParams.output != "-" {
		file, err = os.Create(generateParams.output)
	} else {
		file = os.Stdout
	}
	if err != nil {
		return nil, err
	}
	return file, nil
}

func processFiles(fileChan chan string, wg *sync.WaitGroup) {
	fileList := iradix.New()
	fileListTxn := fileList.Txn()
	count := 0
	for {
		fileEnt, ok := <-fileChan
		if !ok {
			break
		}
		fileListTxn.Insert(convert.UnsafeStringToBytes(strings.TrimPrefix(
			fileEnt, filepath.Clean(generateParams.parentDir)+"/")+"\n"), nil)
		count++
		if count%10000 == 0 {
			log.Printf("Processed count:%d files, last file:%s", count, strings.TrimPrefix(
				fileEnt, filepath.Clean(generateParams.parentDir)+"/"))
		}
	}
	fileList = fileListTxn.Commit()
	iterator := fileList.Root().Iterator()
	file, err := generateParamsToOutputFile()
	if err != nil {
		log.Fatalf("failed to open file:%s err:%s", generateParams.output, err)
	}
	for {
		key, _, ok := iterator.Next()
		if !ok {
			break
		}
		_, err = file.Write(key)
		if err != nil {
			logger.Error("Failed to write file", zap.String("file", file.Name()), zap.Error(err))
		}
	}
	_ = file.Sync()
	_ = file.Close()
	wg.Done()
}

func processDir(dirChan chan *task, fileChan chan string) {

	buffer := make([]byte, 1*1024*1024)
	trees := make([]*iradix.Tree, generateParams.dirRoutines)

	tree := iradix.New()

	for {

		dirTask, ok := <-dirChan
		if !ok {
			return
		}

		childTaskTriggered := false
		iterator := dirTask.iterator()
		for {

			key, _, ok := iterator.Next()
			if !ok {
				break
			}

			// Reset trees
			for i := 0; i < generateParams.dirRoutines; i++ {
				trees[i] = tree
			}

			directory := convert.UnsafeBytesToString(key)

			dirEnts, err := godirwalk.ReadDirents(directory, buffer)
			dirTask.incReadDirOpCount()
			if err != nil {
				// TODO: Record errors on a channel
				logger.Error("ReadDir Failed", zap.Error(err))
				continue
			}

			var dirCount int

			for _, dirEnt := range dirEnts {
				if dirEnt.IsDir() {
					dirCount++
					continue
				}
				if dirEnt.IsRegular() || dirEnt.IsSymlink() {
					fileChan <- directory + "/" + dirEnt.Name()
				}
			}

			if dirCount != 0 {
				childTaskTriggered = true
				// split the child directories into batches
				i := 0
				tasks := dirCount // Min number of sub tasks
				for _, dirEnt := range dirEnts {
					if dirEnt.IsDir() {
						trees[i], _, ok = trees[i].Insert(convert.UnsafeStringToBytes(directory+"/"+dirEnt.Name()), nil)
						if ok {
							logger.Error("Dir already exists", zap.String("dir", directory+"/"+dirEnt.Name()))
						}

						i++
						if i == generateParams.dirRoutines {
							// max tasks = num of dir routines, if dir count > dir routines, batch.
							i = 0
							tasks = generateParams.dirRoutines
						}
					}
				}
				dirTask.incChildTasks(tasks)
				for i := 0; i < tasks; i++ {
					dirChan <- newTask(trees[i], dirTask)
				}
			}
		}
		if !childTaskTriggered {
			dirTask.done()
		}
	}
}

var generateFileListCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a list of files to upload to blobs",
	Long:  "This command takes a parent directory generates a list of all the files. Change current working dir to the top of the tree to be captured for generating relative paths.",
	Run: func(cmd *cobra.Command, args []string) {
		logError := log.New(os.Stderr, "", 0)
		log := log.New(os.Stdout, "", 0)
		count := 0
		file, err := generateParamsToOutputFile()
		if err != nil {
			log.Fatalf("failed to open file:%s err:%s", generateParams.output, err)
		}
		if generateParams.parallel {
			walkDirConcurrently(generateParams.parentDir)
		} else {
			err = godirwalk.Walk(generateParams.parentDir, &godirwalk.Options{
				Callback: func(osPathname string, de *godirwalk.Dirent) error {
					if !de.IsDir() {
						fileToLog := strings.TrimPrefix(osPathname, generateParams.trimPrefix)
						_, err = file.Write(convert.UnsafeStringToBytes(fileToLog + "\n"))
						if err != nil {
							logError.Printf("Failed to write file:%s err:%s", osPathname, err)
						}
					}
					if de.IsSymlink() {
						log.Printf("Skipping sym link:%s", osPathname)
					}
					count++
					if count%10000 == 0 {
						log.Printf("Processed count:%d files, last file:%s", count, osPathname)
					}
					return nil
				},
				ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
					logError.Printf("Hit error:%s path:%s", err, osPathname)
					return godirwalk.SkipNode
				},
				PostChildrenCallback: func(dir string, de *godirwalk.Dirent) error {
					return nil
				},
				ScratchBuffer:       make([]byte, 2*1024*1024),
				FollowSymbolicLinks: true,
				Unsorted:            true, // (optional) set true for faster yet non-deterministic enumeration (see godoc)
			})
			if err != nil {
				fmt.Printf("Failed to start dirWalk error:%s", err)
			}
		}
	},
}

var generateParams struct {
	parentDir       string
	output          string
	trimPrefix      string
	parallel        bool
	dirRoutines     int
	dirChannelSize  int
	fileChannelSize int
}

func init() {
	generateFileListCmd.Flags().StringVarP(&generateParams.parentDir, "parent", "p", "", "Parent directory to process.")
	err := generateFileListCmd.MarkFlagRequired("parent")
	if err != nil {
		log.Fatalln(err)
	}
	generateFileListCmd.Flags().StringVarP(&generateParams.output, "out", "o", "", "Where to write the list.")
	err = generateFileListCmd.MarkFlagRequired("out")
	if err != nil {
		log.Fatalln(err)
	}
	generateFileListCmd.Flags().StringVarP(&generateParams.trimPrefix, "trim-prefix", "t", "", "Remove any prefix from file paths.")
	generateFileListCmd.Flags().BoolVarP(&generateParams.parallel, "parallel", "c", false, "Walk directories in parallel")
	generateFileListCmd.Flags().IntVarP(&generateParams.dirRoutines, "dir", "d", 10, "Concurrency number for directories")
	generateFileListCmd.Flags().IntVarP(&generateParams.dirChannelSize, "dirc", "s", 400000, "Buffer size for directories queue")
	generateFileListCmd.Flags().IntVarP(&generateParams.fileChannelSize, "filec", "f", 400000, "Buffer size for file queue")
	rootCmd.AddCommand(generateFileListCmd)
}
