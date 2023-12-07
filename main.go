package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Package struct {
	Dir         string   // directory containing package sources
	Name        string   // package name
	TestGoFiles []string // test files in package name
}

var (
	userPackages       string
	usePackageTestName bool
	cleanup            bool
	silentFail         bool
)

func main() {
	var packages []Package

	flag.StringVar(&userPackages, "tpkgs", "./...", "packages to run tests")
	flag.BoolVar(&cleanup, "cleanup", false, "clean up dummy tests after usage")
	flag.BoolVar(&usePackageTestName, "pkgnames", false, "use '{package}_test.go' instead of 'dummy_test.go' for dummy test files")
	flag.BoolVar(&silentFail, "sf", false, "silent fail == exit code always 0")
	flag.Parse()

	innerCommand := flag.Args()

	testPackages, isSet := os.LookupEnv("GO_TEST_PACKAGES")
	if isSet {
		userPackages = testPackages
	}

	packages = getPackagesInfo(userPackages)

	createdDummyTests := make([]string, 0)
	for _, p := range packages {
		if len(p.TestGoFiles) == 0 {
			testName := "dummy_test.go"
			if usePackageTestName {
				testName = fmt.Sprintf("%s_test.go", p.Name)
			}
			dummyTestPath := filepath.Join(p.Dir, testName)
			err := ioutil.WriteFile(dummyTestPath, []byte(fmt.Sprintf("package %s", p.Name)), 0644)
			if err != nil {
				writetoStderr(
					fmt.Sprintf(
						"Cannot create dummy test file on path: %s. Cause: %s\n", dummyTestPath, err.Error(),
					),
				)
			} else {
				createdDummyTests = append(createdDummyTests, dummyTestPath)
			}
		}
	}

	if len(innerCommand) > 0 {
		_, exitCode := executeCommand(innerCommand[0], innerCommand[1:], true)
		if exitCode != 0 {
			exitCodeProcessing(exitCode)
		}
	}

	if cleanup {
		cleanCreatedTests(createdDummyTests)
	}
	writeToStdout("===All done!===")
}

func getFormattedPackages(packages string) []string {
	input := strings.TrimSpace(packages)
	return strings.Split(regexp.MustCompile(`\s`).ReplaceAllString(input, " "), " ")
}

func formatOutput(inp []byte) string {
	return strings.TrimSpace(string(inp))
}

func exitCodeProcessing(code int) {
	if !silentFail {
		os.Exit(code)
	} else {
		log.Printf("Potential error code: %d\n", code)
	}
}

func getPackagesInfo(pkgs string) []Package {
	args := []string{"list", "-json"}
	args = append(args, getFormattedPackages(pkgs)...)
	out, exitCode := executeCommand("go", args, false)
	if exitCode != 0 {
		exitCodeProcessing(exitCode)
	}

	dec := json.NewDecoder(bytes.NewReader(out))
	var packages []Package
	for {
		var p Package
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
		}
		packages = append(packages, p)
	}
	return packages
}

func writeToStdout(data string) {
	_, err := os.Stdout.Write([]byte(data + "\n"))
	if err != nil {
		panic("error writing in stdout")
	}
}

func writetoStderr(data string) {
	_, err := os.Stderr.Write([]byte("Error occured: \n" + data + "\n"))
	if err != nil {
		panic("error writing in stderr")
	}
}

func executeCommand(prog string, prArgs []string, outputOnSuccess bool) ([]byte, int) {
	out, err := exec.Command(prog, prArgs...).CombinedOutput()
	if outputOnSuccess {
		writeToStdout(formatOutput(out))
	}

	if err != nil {
		writetoStderr(fmt.Sprintf("Error during executing: %s", prog))
		os.Stderr.Write(out)
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, exitError.ExitCode()
		}
		return nil, 1
	}

	return out, 0
}

func cleanCreatedTests(createdFiles []string) {
	for _, dt := range createdFiles {
		err := os.Remove(dt)
		if err != nil {
			writetoStderr(fmt.Sprintf("Cleanup failed for %s", dt))
		}
	}
}
