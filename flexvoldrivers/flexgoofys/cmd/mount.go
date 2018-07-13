// Copyright © 2018 One Concern

package cmd

import (
	"context"
	"os"
	"strconv"

	"github.com/json-iterator/go"

	goofys "github.com/kahing/goofys/api"
	"github.com/spf13/cobra"
)

type mountOpts struct {
	Bucket   string `json:"bucket"`
	SubPath  string `json:"subPath"`
	DirMode  string `json:"dirMode"`
	FileMode string `json:"fileMode"`
	UID      int    `json:"uid"`
	GID      int    `json:"gid"`
}

// mountCmd represents the mount command
var mountCmd = &cobra.Command{
	Use:   "mount [MOUNT DIR] [JSON OPTIONS]",
	Short: "mounts a S3 bucket to a local folder",
	Long:  `mounts a S3 bucket to a local folder`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		mpath := args[0]
		if mpath == "" {
			respond(dsFailure, "mount path is required")
		}

		var opts mountOpts
		if err := jsoniter.UnmarshalFromString(args[1], &opts); err != nil {
			respond(dsFailure, err.Error())
		}

		cfg, err := goofysConfig(opts)
		if err != nil {
			respond(dsFailure, err.Error())
		}
		_, _, err = goofys.Mount(context.Background(), opts.Bucket, cfg)
		if err != nil {
			respond(dsFailure, err.Error())
		}

		respond(dsSuccess, "Bucket was mounted.")
	},
}

func init() {
	rootCmd.AddCommand(mountCmd)

}

func goofysConfig(opts mountOpts) (*goofys.Config, error) {
	var res goofys.Config
	res.Foreground = true

	dm, err := dirMode(opts.DirMode)
	if err != nil {
		return nil, err
	}
	res.DirMode = dm

	fm, err := fileMode(opts.FileMode)
	if err != nil {
		return nil, err
	}
	res.FileMode = fm

	return &res, nil
}

func dirMode(dm string) (os.FileMode, error) {
	if dm == "" {
		return 0755, nil
	}
	res, err := strconv.ParseUint(dm, 8, 32)
	if err != nil {
		return 0, err
	}
	return os.FileMode(res), nil
}

func fileMode(dm string) (os.FileMode, error) {
	if dm == "" {
		return 0644, nil
	}
	res, err := strconv.ParseUint(dm, 8, 32)
	if err != nil {
		return 0, err
	}
	return os.FileMode(res), nil
}
