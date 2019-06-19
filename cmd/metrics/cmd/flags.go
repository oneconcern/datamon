// Copyright © 2018 One Concern

package cmd

import (
	"github.com/spf13/cobra"
)

type paramsT struct {
	root struct {
		cpuProfPath string
		memProfPath string
	}
	upload struct {
		max       int
		numFiles  int
		numChunks int
		fileType  string
	}
}

var params paramsT = paramsT{}

func addCpuProfPath(cmd *cobra.Command) string {
	const cpuprof = "cpuprof"
	cmd.Flags().StringVar(&params.root.cpuProfPath, cpuprof, "cpu.prof", "The path to output the pprof cpu information.")
	return cpuprof
}

func addMemProfPath(cmd *cobra.Command) string {
	const memprof = "memprof"
	cmd.Flags().StringVar(&params.root.memProfPath, memprof, "mem.prof", "The path to output the pprof mem information.")
	return memprof
}

func addUploadFilesize(cmd *cobra.Command) string {
	const filesize = "filesize"
	cmd.Flags().IntVar(&params.upload.max, filesize, 16, "File size (approx MiB) to upload")
	return filesize
}

func addUploadNumFiles(cmd *cobra.Command) string {
	const numFiles = "num-files"
	cmd.Flags().IntVar(&params.upload.numFiles, numFiles, 40, "Number of files to upload")
	return numFiles
}

func addUploadNumChunks(cmd *cobra.Command) string {
	const numChunks = "num-chunks"
	cmd.Flags().IntVar(&params.upload.numChunks, numChunks, 40, "Number of chunks to upload in case of chunked file")
	return numChunks
}

func addUploadFileType(cmd *cobra.Command) string {
	const fileType = "file-type"
	cmd.Flags().StringVar(&params.upload.fileType, fileType, "stripe",
		"type of file to upload among 'chunks', 'stripe', 'rand'")
	return fileType
}
