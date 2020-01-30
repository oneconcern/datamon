package cmd

import (
	"fmt"
	stdlog "log"
	"os"

	"github.com/oneconcern/datamon/pkg/auth"
)

var (
	// control over stdout/stderr
	log    = stdlog.New(os.Stdout, "", 0)
	errlog = stdlog.New(os.Stderr, "ERROR:", 0)

	// infoLogger wraps informative messages to os.Stderr without cluttering expected output in tests.
	infoLogger = stdlog.New(os.Stderr, "INFO:", 0)

	// global used to patch over calls to os.Exit() during tests
	osExit = os.Exit

	// global used to patch over calls to Authable.Principal() during tests
	authorizer auth.Authable
)

func wrapFatalln(msg string, err error) {
	if err == nil {
		errlog.Fatalln(msg)
	} else {
		errlog.Fatalf("%v", fmt.Errorf(msg+": %w", err))
	}
}

// wrapFatalWithCodef is equivalent to log.Fatalf but controls the exit code returned to the command
func wrapFatalWithCodef(code int, format string, args ...interface{}) {
	errlog.Printf(format, args...)
	osExit(code)
}
