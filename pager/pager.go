// The pager package allows the program to easily pipe it's
// standard output through a pager program
// (like how the man command does).
//
// Borrowed from: https://gist.github.com/dchapes/1d0c538ce07902b76c75 and
// reworked slightly.

package pager

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
)

var pager struct {
	cmd  *exec.Cmd
	file io.WriteCloser
}

// The environment variables to check for the name of (and arguments to)
// the pager to run.
var PagerEnvVariables = []string{"PAGER"}

// The command names in $PATH to look for if none of the environment
// variables are set.
// Cannot include arguments.
var PagerCommands = []string{"less", "more"}

func pagerExecPath() (pagerPath string, args []string, err error) {
	for _, testVar := range PagerEnvVariables {
		pagerPath = os.Getenv(testVar)
		if pagerPath != "" {
			args = strings.Fields(pagerPath)
			if len(args) > 1 {
				return args[0], args[1:], nil
			}
		}
	}

	// This default only gets used if PagerCommands is empty.
	err = exec.ErrNotFound
	for _, testPath := range PagerCommands {
		pagerPath, err = exec.LookPath(testPath)
		if err == nil {
			switch {
			case path.Base(pagerPath) == "less":
				// TODO(seanc@): Make the buffer size conditional
				args := []string{"-X", "-F", "-R", "--buffers=65535"}
				return pagerPath, args, nil
			default:
				return pagerPath, nil, nil
			}
		}
	}
	return "", nil, err
}

// New returns a new io.WriteCloser connected to a pager.
// The returned WriteCloser can be used as a replacement to os.Stdout,
// everything written to it is piped to a pager.
// To determine what pager to run, the environment variables listed
// in PagerEnvVariables are checked.
// If all are empty/unset then the commands listed in PagerCommands
// are looked for in $PATH.
func New() (io.WriteCloser, error) {
	if pager.cmd != nil {
		return nil, errors.New("pager: already exists")
	}
	pagerPath, args, err := pagerExecPath()
	if err != nil {
		return nil, err
	}

	// If the pager is less(1), set some useful options
	switch {
	case path.Base(pagerPath) == "less":
		os.Setenv("LESSSECURE", "1")
	}

	pager.cmd = exec.Command(pagerPath, args...)
	pager.cmd.Stdout = os.Stdout
	pager.cmd.Stderr = os.Stderr
	w, err := pager.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	f, ok := w.(io.WriteCloser)
	if !ok {
		return nil, errors.New("pager: exec.Command.StdinPipe did not return type io.WriteCloser")
	}
	pager.file = f
	err = pager.cmd.Start()
	if err != nil {
		return nil, err
	}
	return pager.file, nil
}

// Wait closes the pipe to the pager setup with New() or Stdout() and waits
// for it to exit.
//
// This should normally be called before the program exists,
// typically via a defer call in main().
func Wait() {
	if pager.cmd == nil {
		return
	}
	pager.file.Close()
	pager.cmd.Wait()
}
