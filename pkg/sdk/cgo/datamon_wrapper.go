package main

import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/oneconcern/datamon/cmd/datamon/cmd"
	"github.com/oneconcern/datamon/pkg/core"
	"github.com/oneconcern/datamon/pkg/dlogger"
	"github.com/oneconcern/datamon/pkg/model"
	"github.com/oneconcern/datamon/pkg/storage/gcs"
	"gopkg.in/yaml.v2"
)

// Config is the same as the CLI config
type Config = cmd.CLIConfig

func errToHost(err error, erro **C.Char) int {
	if erro != nil {
		*erro = C.CString(err.Error())
	}
	return -1
}

func wrapErrToHost(msg string, err error, erro **C.Char) int {
	return errToHost(fmt.Sprintf("ERROR: %s: %v", msg, err), erro)
}

//export listRepos
func listRepos(config *C.Char, output **C.char, erro **C.Char) int {
	if output == nil {
		return errToHost(fmt.Errorf("output arg required to be not nil"), erro)
	}
	if config == nil {
		// TODO(fred): should fall back to file, env, etc
		return errToHost(fmt.Errorf("config arg required to be not nil"), erro)
	}

	var config Config
	err := json.Unmarshal([]byte(C.GoString(config)), &config)
	if err != nil {
		return errToHost(err, erro)
	}

	// TODO(fred): factorize with cli
	if config.Credential == "" {
		config.Credential = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
	logger := dlogger.MustGetLogger("info")
	configStore, err := gcs.New(context.Background(), flags.core.Config, config.Credential, gcs.Logger(logger))
	if err != nil {
		return wrapErrToHost("failed to get config store", err, error)
	}

	rdr, err := handleContextErr(configStore.Get(context.Background(), model.GetPathToContext(flags.context.Descriptor.Name)))
	if err != nil {
		return wrapErrToHost("failed to get context details from config store for context ", err, erro)
	}
	b, err := ioutil.ReadAll(rdr)
	if err != nil {
		return wrapErrToHost("failed to read context details", err, err)
	}
	contextDescriptor := model.Context{}
	err = yaml.Unmarshal(b, &contextDescriptor)
	if err != nil {
		wrapFatalln("failed to unmarshal", err)
		return
	}
	config.populateRemoteConfig(&datamonFlags)

	remoteStores, err := paramsToDatamonContext(ctx, datamonFlags)
	if err != nil {
		return errToHost(err, erro)
		return wrapErrToHost("create remote stores", err)
	}

	repos, err = core.ListRepos(remoteStores,
		core.ConcurrentList(100),
		core.BatchSize(1024),
	)
	if err != nil {
		return wrapErrToHost("download repo list", err)
	}
	out, err := json.Marshal(repos)

	return 0
}

/*
func buildRequirements(requirementsAsJSON *C.char) (*authorizer.Requirements, error) {
	if requirementsAsJSON == nil {
		return nil, nil
	}
	var requirements authorizer.Requirements
	err := json.Unmarshal([]byte(C.GoString(requirementsAsJSON)), &requirements)
	return &requirements, err
}

//export sdkAuthAllowed
func sdkAuthAllowed(user, token, requirementsAsJSON *C.char, erro **C.char) int {
	// sdkAuthAllowed is a C wrapper for authorizer's Allowed()
	if auth != nil {
		requirements, err := buildRequirements(requirementsAsJSON)
		r, err := auth.AllowedE(C.GoString(user), C.GoString(token), requirements)
		if err != nil {
			// malloc
			if erro != nil {
				*erro = C.CString(err.Error())
			}
			return -1
		}
		if r {
			return 1
		}
		return 0
	}
	if erro != nil {
		*erro = C.CString("a new authorizer must be instantiated first")
	}
	return -1
}

//export sdkAuthAuthorized
func sdkAuthAuthorized(token *C.char, erro **C.char) *C.char {
	// sdkAuthAuthorized is a C wrapper for authorizer's Authorized()
	if auth != nil {
		p, err := auth.Authorized(C.GoString(token))
		var subject string

		if err != nil {
			// malloc
			if erro != nil {
				*erro = C.CString(err.Error())
			}
			return nil
		}
		subject = p.Username()
		// malloc
		return C.CString(subject)
	}
	if erro != nil {
		*erro = C.CString("a new authorizer must be instantiated first")
	}
	return nil
}

//export sdkAuthCheckGlobalRequirements
func sdkAuthCheckGlobalRequirements(user, token *C.char, erro **C.char) int {
	// sdkAuthCheckGlobalRequirements is a C wrapper for authorizer's CheckGlobalRequirements()
	if auth != nil {
		r, err := auth.CheckGlobalRequirements(C.GoString(user), C.GoString(token))
		if err != nil {
			// malloc
			if erro != nil {
				*erro = C.CString(err.Error())
			}
			return -1
		}
		if r {
			return 1
		}
		return 0
	}
	if erro != nil {
		*erro = C.CString("a new authorizer must be instantiated first")
	}
	return -1
}

//export sdkAuthInitAuthorizer
func sdkAuthInitAuthorizer() {
	// empty init entry point at the moment
}

//export sdkAuthNewAuthorizer
func sdkAuthNewAuthorizer(args *C.char, erro **C.char) int {
	// sdkAuthNewAuthorizer creates a new authorizer with a custom SDK config as JSON
	// if no config given, defaults to env/flags parsing
	var config *authorizer.Config
	var err error
	if args != nil {
		err = json.Unmarshal([]byte(C.GoString(args)), config)
		if err != nil {
			if erro != nil {
				*erro = C.CString(err.Error())
			}
			return -1
		}
	}
	auth, err = authorizer.NewAuthorizer(config)
	if err != nil {
		if erro != nil {
			*erro = C.CString(err.Error())
		}
		return -1
	}
	return 0
}
*/

func main() {
}
