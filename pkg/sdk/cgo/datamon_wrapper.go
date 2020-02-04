package main

import "C"

import (
	"encoding/json"
	"fmt"

	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/sdk/cgo/crap"
)

// TODO(fred): use jsoniter instead of stdlib encoding/json

// errToHost returns some errno to the host program.
//
// NOTE: freeing the allocated memory is the responsibility of the caller.
func errToHost(err error, errno **C.char) int {
	if errno != nil {
		// malloc: let the caller handle free
		*errno = C.CString(err.Error())
	}
	return -1
}

func wrapErrToHost(msg string, err error, errno **C.char) int {
	return errToHost(fmt.Errorf("ERROR: %s: %v", msg, err), errno)
}

//export listRepos
func listRepos(jsonConfig *C.char, jsonOutput **C.char, errno **C.char) int {
	if jsonOutput == nil {
		return errToHost(fmt.Errorf("output arg required to be not nil"), errno)
	}
	if jsonConfig == nil {
		// TODO(fred): should fall back to file, env, etc
		return errToHost(fmt.Errorf("config arg required to be not nil"), errno)
	}

	config, err := crap.ParseConfigAndFlagsEtc([]byte(C.GoString(jsonConfig)))
	if err != nil {
		return errToHost(err, errno)
	}

	_, remoteStores, _, err := crap.SetupStoresEtc(config)
	if err != nil {
		return errToHost(err, errno)
	}

	repos, err := core.ListRepos(remoteStores,
		core.ConcurrentList(100),
		core.BatchSize(1024),
	)
	if err != nil {
		return wrapErrToHost("download repo list", err, errno)
	}

	out, err := json.Marshal(repos)
	if err != nil {
		return wrapErrToHost("marshal repo list", err, errno)
	}

	// malloc: let the caller handle free
	*jsonOutput = C.CString(string(out))
	return 0
}

//export listBundles
func listBundles(jsonConfig *C.char, repo *C.char, jsonOutput **C.char, errno **C.char) int {
	if jsonOutput == nil {
		return errToHost(fmt.Errorf("output arg required to be not nil"), errno)
	}
	if jsonConfig == nil {
		// TODO(fred): should fall back to file, env, etc
		return errToHost(fmt.Errorf("config arg required to be not nil"), errno)
	}

	config, err := crap.ParseConfigAndFlagsEtc([]byte(C.GoString(jsonConfig)))
	if err != nil {
		return errToHost(err, errno)
	}

	_, remoteStores, _, err := crap.SetupStoresEtc(config)
	if err != nil {
		return errToHost(err, errno)
	}

	bundles, err := core.ListBundles(C.GoString(repo), remoteStores,
		core.ConcurrentList(100),
		core.BatchSize(1024),
	)
	if err != nil {
		return wrapErrToHost("download bundle list", err, errno)
	}

	out, err := json.Marshal(bundles)
	if err != nil {
		return wrapErrToHost("marshal bundle list", err, errno)
	}

	// malloc: let the caller handle free
	*jsonOutput = C.CString(string(out))
	return 0
}

func main() {
}
