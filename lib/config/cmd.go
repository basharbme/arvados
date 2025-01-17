// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

var DumpCommand dumpCommand

type dumpCommand struct{}

func (dumpCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	defer func() {
		if err != nil {
			fmt.Fprintf(stderr, "%s\n", err)
		}
	}()

	loader := &Loader{
		Stdin:  stdin,
		Logger: ctxlog.New(stderr, "text", "info"),
	}

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)
	loader.SetupFlags(flags)

	err = flags.Parse(args)
	if err == flag.ErrHelp {
		err = nil
		return 0
	} else if err != nil {
		return 2
	}

	if len(flags.Args()) != 0 {
		flags.Usage()
		return 2
	}

	cfg, err := loader.Load()
	if err != nil {
		return 1
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return 1
	}
	_, err = stdout.Write(out)
	if err != nil {
		return 1
	}
	return 0
}

var CheckCommand checkCommand

type checkCommand struct{}

func (checkCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	var logbuf = &bytes.Buffer{}
	defer func() {
		io.Copy(stderr, logbuf)
		if err != nil {
			fmt.Fprintf(stderr, "%s\n", err)
		}
	}()

	logger := logrus.New()
	logger.Out = logbuf
	loader := &Loader{
		Stdin:  stdin,
		Logger: logger,
	}

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)
	loader.SetupFlags(flags)

	err = flags.Parse(args)
	if err == flag.ErrHelp {
		err = nil
		return 0
	} else if err != nil {
		return 2
	}

	if len(flags.Args()) != 0 {
		flags.Usage()
		return 2
	}

	// Load the config twice -- once without loading deprecated
	// keys/files, once with -- and then compare the two resulting
	// configs. This reveals whether the deprecated keys/files
	// have any effect on the final configuration.
	//
	// If they do, show the operator how to update their config
	// such that the deprecated keys/files are superfluous and can
	// be deleted.
	loader.SkipDeprecated = true
	loader.SkipLegacy = true
	withoutDepr, err := loader.Load()
	if err != nil {
		return 1
	}
	loader.SkipDeprecated = false
	loader.SkipLegacy = false
	withDepr, err := loader.Load()
	if err != nil {
		return 1
	}
	cmd := exec.Command("diff", "-u", "--label", "without-deprecated-configs", "--label", "relying-on-deprecated-configs", "/dev/fd/3", "/dev/fd/4")
	for _, obj := range []interface{}{withoutDepr, withDepr} {
		y, _ := yaml.Marshal(obj)
		pr, pw, err := os.Pipe()
		if err != nil {
			return 1
		}
		defer pr.Close()
		go func() {
			io.Copy(pw, bytes.NewBuffer(y))
			pw.Close()
		}()
		cmd.ExtraFiles = append(cmd.ExtraFiles, pr)
	}
	diff, err := cmd.CombinedOutput()
	if bytes.HasPrefix(diff, []byte("--- ")) {
		fmt.Fprintln(stdout, "Your configuration is relying on deprecated entries. Suggest making the following changes.")
		stdout.Write(diff)
		err = nil
		return 1
	} else if len(diff) > 0 {
		fmt.Fprintf(stderr, "Unexpected diff output:\n%s", diff)
		return 1
	} else if err != nil {
		return 1
	}
	if logbuf.Len() > 0 {
		return 1
	}
	return 0
}

var DumpDefaultsCommand defaultsCommand

type defaultsCommand struct{}

func (defaultsCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	_, err := stdout.Write(DefaultYAML)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
