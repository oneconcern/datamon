package main

import (
	"fmt"
	"log"

	p "github.com/oneconcern/datamon/pkg/sidecar/param"
)

func noError(err error, msg string) {
	if err == nil {
		return
	}
	fmt.Println(msg)
	log.Fatal(err)
}

func main() {
	fuseParams, err := p.NewFUSEParams(
		p.FUSECoordPoint("/tmp/coord"),
		p.FUSEConfigBucketName("datamon-config-test-sdjfhga"),
		p.FUSEContextName("datamon-sidecar-test"),
	)
	noError(err, "init postgres params")
	err = fuseParams.AddBundle(
		p.BDName("src"),
		p.BDSrcByLabel(
			"/tmp/mount",
			"ransom-datamon-test-repo",
			"testlabel",
		),
	)
	noError(err, "add src bundle")
	err = fuseParams.AddBundle(
		p.BDName("dest"),
		p.BDDest("ransom-datamon-test-repo", "result of container coordination demo", "/tmp/upload"),
		p.BDDestLabel("coordemo"),
		p.BDDestBundleIDFile("/tmp/bundleid.txt"),
	)
	noError(err, "add destination bundle")
	// NB.  this is the kind of thing i'd like to disallow before making the api public.
	//  the it shouldn't be possible to mutate the unadorned data directly.
	fuseParams.Globals.SleepInsteadOfExit = true
	err = fuseParams.FirstCutSidecarFmt("hack/fuse-demo/gen/fuse-params.yaml")
	noError(err, "write out sidecar serialization")
}
