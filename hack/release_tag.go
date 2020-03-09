// +build ignore

package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"

	"gotest.tools/icmd"
)

const (
	latestUnofficial     = "latest-unofficial"
	hashTag              = "hash-%s"
	latestOfficiaRebuild = "latest-official-rebuild"
	latestOfficialInit   = "latest-official-init"
)

var (
	versionRe, hashRe *regexp.Regexp
	args              struct {
		isLatest bool
		image    string
	}
)

func init() {
	versionRe = regexp.MustCompile(`(?m)^v.*`)
	hashRe = regexp.MustCompile(`(?m)^commit\s+(.*?)\s`)
}

func main() {
	flag.BoolVar(&args.isLatest, "l", false, "makes latest status tag")
	flag.StringVar(&args.image, "i", "", "image tag to check when latest (tags latest-official-rebuild, latest-official-init)")
	flag.Parse()
	fmt.Println(getVersion())
}

func getVersion() string {
	tags := getTag()
	switch len(tags) {
	case 0:
		if args.isLatest {
			return latestUnofficial
		}
		return fmt.Sprintf(hashTag, getHash())
	case 1:
		if !args.isLatest {
			return tags[0][0]
		}
		if checkRepo(tags[0][0]) {
			return latestOfficiaRebuild
		}
		return latestOfficialInit
	default:
		log.Fatalf("ambiguous tags at HEAD: multiple tags begin with 'v': %v", tags)
	}
	return ""
}

func getTag() [][]string {
	resTag := icmd.RunCommand("git", "tag", "--points-at", "HEAD")
	if resTag.ExitCode != 0 {
		return nil
	}
	return versionRe.FindAllStringSubmatch(resTag.Stdout(), -1)
}

func getHash() string {
	resHash := icmd.RunCommand("git", "show", "--abbrev-commit", "--quiet")
	if resHash.ExitCode != 0 {
		log.Fatalf("cannot execute git show")
	}
	hashes := hashRe.FindAllStringSubmatch(resHash.Stdout(), -1)
	if len(hashes) == 0 || len(hashes[0]) < 2 {
		log.Fatalf("expected a commit hash in: %q", resHash.Stdout())
	}
	return hashes[0][1]
}

func checkRepo(tag string) bool {
	if args.image == "" {
		return false
	}
	resDocker := icmd.RunCommand("docker", "pull", "--quiet", fmt.Sprintf("%s:%s", args.image, tag))
	return resDocker.ExitCode == 0
}
