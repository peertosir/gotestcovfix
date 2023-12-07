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
	TestGoFiles []string // tedt files in package name
}

func main() {
	var packages []Package
	var userPackages string
	var usePackageTestName bool
	var cleanup bool

	flag.StringVar(&userPackages, "tpkgs", "./...", "packages to run tests")
	flag.BoolVar(&cleanup, "cleanup", false, "clean up dummy tests after usage")
	flag.BoolVar(&usePackageTestName, "pkgnames", false, "use '{package}_test.go' instead of 'dummy_test.go' for dummy test files")
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
				log.Printf("Cannot create dummy test file on path: %s. Cause: %s\n", dummyTestPath, err.Error())
			} else {
				createdDummyTests = append(createdDummyTests, dummyTestPath)
			}
		}
	}

	if len(innerCommand) > 0 {
		out, err := exec.Command(innerCommand[0], innerCommand[1:]...).CombinedOutput()
		if err != nil {
			cleanCreatedTests(createdDummyTests)
			log.Fatalf("Error occured: %s, %s\n", formatOutput(out), err.Error())
		}
		_, err = os.Stdout.Write(out)
		if err != nil {
			cleanCreatedTests(createdDummyTests)
			log.Fatalf("Error during writing to stdin: %s\n", err.Error())
		}
	}

	if cleanup {
		cleanCreatedTests(createdDummyTests)
	}
	fmt.Println("All done!")
}

func getFormattedPackages(packages string) []string {
	input := strings.TrimSpace(packages)
	return strings.Split(regexp.MustCompile(`\s`).ReplaceAllString(input, " "), " ")
}

func formatOutput(inp []byte) string {
	return strings.TrimSpace(string(inp))
}

func getPackagesInfo(pkgs string) []Package {
	command := []string{"list", "-json"}
	command = append(command, getFormattedPackages(pkgs)...)
	out, err := exec.Command("go", command...).CombinedOutput()
	if err != nil {
		log.Fatalf("Error during packages acquiring: %s, %s", formatOutput(out), err.Error())
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

func cleanCreatedTests(createdFiles []string) {
	for _, dt := range createdFiles {
		err := os.Remove(dt)
		if err != nil {
			log.Printf("Cleanup failed for %s", dt)
		}
	}
}
