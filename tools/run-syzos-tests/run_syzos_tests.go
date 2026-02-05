// Copyright 2026 syzkaller authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	var (
		flagProg2c  = flag.String("prog2c", "bin/syz-prog2c", "Path to syz-prog2c binary")
		flagTimeout = flag.Int("timeout", 60, "Test execution timeout in seconds")
		flagTest    = flag.String("test", "", "Single test file to run")
		flagRunner  = flag.String("runner", "", "Runner command to execute the test binary")
	)
	flag.Parse()

	var tests []string
	var err error

	if *flagTest != "" {
		tests = []string{*flagTest}
	} else {
		if len(flag.Args()) != 1 {
			fmt.Fprintf(os.Stderr, "Usage: %s [flags] <test_list_file>\n", os.Args[0])
			flag.PrintDefaults()
			os.Exit(1)
		}
		testListFile := flag.Args()[0]
		tests, err = readTests(testListFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading test list: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Loaded %d tests from %s\n", len(tests), testListFile)
	}

	if _, err := os.Stat(*flagProg2c); err != nil {
		fmt.Fprintf(os.Stderr, "Error: syz-prog2c binary not found at %s. Please build it first.\n", *flagProg2c)
		os.Exit(1)
	}

	passed := 0
	failed := 0

	for _, test := range tests {
		if _, err := os.Stat(test); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Test file %s not found.\n", test)
			os.Exit(1)
		}
	}

	timeout := time.Duration(*flagTimeout) * time.Second

	for _, test := range tests {
		if runTest(test, *flagProg2c, *flagRunner, timeout) {
			passed++
		} else {
			failed++
		}
	}

	fmt.Println("--------------------")
	fmt.Printf("Total: %d\n", len(tests))
	fmt.Printf("PASS: %d\n", passed)
	fmt.Printf("FAIL: %d\n", failed)

	if failed > 0 {
		os.Exit(1)
	}
}

func readTests(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tests []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.IndexByte(line, '#'); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line != "" {
			tests = append(tests, line)
		}
	}
	return tests, scanner.Err()
}

func runTest(testPath, prog2cBin, runner string, timeout time.Duration) bool {
	fmt.Printf("Running %s: ", testPath)

	const (
		reproC   = "repro.c"
		reproBin = "./repro"
	)

	// Clean up previous run
	os.Remove(reproC)
	os.Remove(reproBin)
	/*
	defer func() {
		os.Remove(reproC)
		os.Remove(reproBin)
	}()
	*/

	progFile := testPath

	// Run syz-prog2c and capture stdout to file.
	f, err := os.Create(reproC)
	if err != nil {
		fmt.Printf("FAIL (create repro.c: %v)\n", err)
		return false
	}
	prog2cCmd := exec.Command(prog2cBin, "-prog", progFile)
	prog2cCmd.Stdout = f
	if err := prog2cCmd.Run(); err != nil {
		f.Close()
		fmt.Printf("FAIL (syz-prog2c: %v)\n", err)
		return false
	}
	f.Close()

	// Compile the reproducer with -w to suppress warnings.
	gccCmd := exec.Command("gcc", reproC, "-static", "-o", reproBin, "-w")
	if out, err := gccCmd.CombinedOutput(); err != nil {
		fmt.Printf("FAIL (gcc: %v)\n%s\n", err, out)
		return false
	}

	// Run.
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmdName string
	var cmdArgs []string

	if runner != "" {
		cmdName = runner
		cmdArgs = []string{reproBin}
	} else {
		cmdName = reproBin
	}

	runCmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	var output bytes.Buffer
	runCmd.Stdout = &output
	runCmd.Stderr = &output

	err = runCmd.Run()
	duration := time.Since(start)

	if ctx.Err() == context.DeadlineExceeded {
		fmt.Printf("TIMEOUT (%v)\n", timeout)
		return false
	}

	// Compare output.
	outStr := strings.TrimSpace(output.String())
	if outStr == "executing program" {
		fmt.Printf("PASS (%.2fs)\n", duration.Seconds())
		return true
	}

	fmt.Printf("FAIL (output mismatch or error)\n--- Output ---\n%s\n--------------\n", outStr)
	return false
}
