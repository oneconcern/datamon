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
		max       float64
		numFiles  int
		numChunks int
		fileType  string
		mockDest  bool
	}
}

var params paramsT = paramsT{}

func addCpuProfPath(cmd *cobra.Command) string {
	const flagName = "cpuprof"
	cmd.Flags().StringVar(&params.root.cpuProfPath, flagName, "cpu.prof",
		"The path to output the pprof cpu information.")
	return flagName
}

func addMemProfPath(cmd *cobra.Command) string {
	const flagName = "memprof"
	cmd.Flags().StringVar(&params.root.memProfPath, flagName, "mem.prof",
		"The path to output the pprof mem information.")
	return flagName
}

func addUploadFilesize(cmd *cobra.Command) string {
	const flagName = "filesize"
	cmd.Flags().Float64Var(&params.upload.max, flagName, 16,
		"File size (approx MiB) to upload")
	return flagName
}

func addUploadNumFiles(cmd *cobra.Command) string {
	const flagName = "num-files"
	cmd.Flags().IntVar(&params.upload.numFiles, flagName, 40,
		"Number of files to upload")
	return flagName
}

func addUploadNumChunks(cmd *cobra.Command) string {
	const flagName = "num-chunks"
	cmd.Flags().IntVar(&params.upload.numChunks, flagName, 40,
		"Number of chunks to upload in case of chunked file")
	return flagName
}

func addUploadFileType(cmd *cobra.Command) string {
	const flagName = "file-type"
	cmd.Flags().StringVar(&params.upload.fileType, flagName, "stripe",
		"type of file to upload among 'chunks', 'stripe', 'rand'")
	return flagName
}

func addUploadMockDest(cmd *cobra.Command) string {
	const flagName = "mock-dest"
	cmd.Flags().BoolVar(&params.upload.mockDest, flagName, true,
		"whether to use GCS or a mock/stub/spy storage.Store")
	return flagName
}
