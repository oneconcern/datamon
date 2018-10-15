// Copyright © 2018 One Concern

package cmd

import (
	"fmt"
	"log"

	"github.com/oneconcern/pipelines/pkg/cli/cflags"
	"github.com/oneconcern/datamon/pkg/fingerprint"
	"github.com/spf13/cobra"
)

var checksumOpts csOpts

type csOpts struct {
	Size     cflags.ByteSize
	LeafSize cflags.ByteSize
}

// checksumCmd represents the checksum command
var checksumCmd = &cobra.Command{
	Use:   "checksum",
	Short: "Create a blake2b checksum for a file or a tree of files",
	Long:  ``,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fp, err := fingerprint.New(
			fingerprint.Size(uint8(int64(checksumOpts.Size))),
			fingerprint.LeafSize(int64(checksumOpts.LeafSize)),
		).Process(args[0])

		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%x\n", fp)
	},
}

func init() {
	fileCmd.AddCommand(checksumCmd)

	fls := checksumCmd.Flags()
	checksumOpts.Size = cflags.ByteSize(64)
	checksumOpts.LeafSize = cflags.ByteSize(5 * 1048576)
	fls.Var(&checksumOpts.Size, "digest-size", "Digest size in bytes")
	fls.Var(&checksumOpts.LeafSize, "leaf-size", "Leaf size in bytes for tree mode")

	for i := 1; i < 10; i++ {
		checksumCmd.MarkZshCompPositionalArgumentFile(i, "*")
	}
}
