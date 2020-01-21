package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/oneconcern/datamon/pkg/auth"
)

var (
	// globals used to patch over calls to os.Exit() during test

	logFatalln = log.Fatalln
	logFatalf  = log.Fatalf
	osExit     = os.Exit

	// used to patch over calls to Authable.Principal() during test
	authorizer auth.Authable

	// infoLogger wraps informative messages to os.Stdout without cluttering expected output in tests.
	// To be used instead on fmt.Printf(os.Stdout, ...)
	infoLogger = log.New(os.Stdout, "", 0)
	logStdOut  = fmt.Printf
)

func wrapFatalln(msg string, err error) {
	if err == nil {
		logFatalln(msg)
	} else {
		logFatalf("%v", fmt.Errorf(msg+": %w", err))
	}
}

func wrapFatalWithCodef(code int, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	osExit(code)
}
