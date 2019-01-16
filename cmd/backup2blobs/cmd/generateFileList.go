package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/karrick/godirwalk"
	"github.com/spf13/cobra"
)

var generateFileListCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a list of files to upload to blobs",
	Long:  "This command takes a parent directory generates a list of all the files. Change current working dir to the top of the tree to be captured for generating relative paths.",
	Run: func(cmd *cobra.Command, args []string) {
		logError := log.New(os.Stderr, "", 0)
		log := log.New(os.Stdout, "", 0)
		count := 0
		file, err := os.Create(generateParams.output)
		if err != nil {
			log.Fatalf("failed to open file:%s err:%s", generateParams.output, err)
		}
		err = godirwalk.Walk(generateParams.parentDir, &godirwalk.Options{
			Callback: func(osPathname string, de *godirwalk.Dirent) error {
				if !de.IsDir() {
					fileToLog := strings.TrimPrefix(osPathname, generateParams.trimPrefix)
					_, err := file.Write([]byte(fileToLog + "\n"))
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
				return godirwalk.Halt
			},
			PostChildrenCallback: func(dir string, de *godirwalk.Dirent) error {
				log.Printf("Finished process dir:%s count:%d", dir, count)
				return nil
			},
			FollowSymbolicLinks: true,
			Unsorted:            true, // (optional) set true for faster yet non-deterministic enumeration (see godoc)
		})
		if err != nil {
			fmt.Printf("Failed to start dirWalk error:%s", err)
		}
	},
}

var generateParams struct {
	parentDir  string
	output     string
	trimPrefix string
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
	err = generateFileListCmd.MarkFlagRequired("out")
	if err != nil {
		log.Fatalln(err)
	}
	rootCmd.AddCommand(generateFileListCmd)
}
