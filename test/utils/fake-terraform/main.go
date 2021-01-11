// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	// expectedExitCodes is a list of expected exit codes for the different commands in form `42` or `init=0,apply=42`.
	expectedExitCodes string
	sleepDuration     string
)

// This packages contains a simple program which can be built in tests to mock terraform executions
// It basically just writes some lines to stdout and stderr, sleeps for `sleepDuration` and exits with `exitCode`.
func main() {
	fmt.Println("some terraform output")
	fmt.Println("args: " + strings.Join(os.Args[1:], " "))

	exitCode := getExpectedExitCode()

	if sleepDuration != "" && (len(os.Args) <= 1 || os.Args[1] != "init") {
		done := make(chan struct{})
		defer close(done)

		// setup signal handler
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			select {
			case s := <-sigCh:
				fmt.Printf("fake terraform received signal: %s\n", s.String())
			case <-done:
			}
		}()

		// sleep for specified duration
		duration, err := time.ParseDuration(sleepDuration)
		if err != nil {
			panic(err)
		}

		fmt.Printf("doing some long running IaaS ops for %s\n", duration.String())
		time.Sleep(duration)
	}

	fmt.Println("finished terraform execution")
	_, _ = fmt.Fprintln(os.Stderr, "some terraform error")

	os.Exit(exitCode)
}

func getExpectedExitCode() int {
	if expectedExitCodes == "" {
		return 0
	}
	if !strings.Contains(expectedExitCodes, ",") {
		code, err := strconv.Atoi(expectedExitCodes)
		if err != nil {
			panic(err)
		}
		return code
	}

	exitCodes := strings.Split(expectedExitCodes, ",")
	if len(exitCodes) == 0 || len(os.Args) <= 1 || os.Args[1] == "" {
		return 0
	}

	command := os.Args[1]
	for _, e := range exitCodes {
		if strings.HasPrefix(e, command+"=") {
			code, err := strconv.Atoi(strings.TrimPrefix(e, command+"="))
			if err != nil {
				panic(err)
			}
			return code
		}
	}
	return 0
}
