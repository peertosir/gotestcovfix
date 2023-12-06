package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	TestGoFiles []string // package name
}

func main() {
	var packages []Package
	var inputPackages string
	var usePackageTestName bool
	var cleanup bool

	flag.StringVar(&inputPackages, "tpkgs", "./...", "packages to run tests")
	flag.BoolVar(&cleanup, "cleanup", false, "clean up dummy tests after usage")
	flag.BoolVar(&usePackageTestName, "pkgtestname", false, "use 'pkg_test.go' instead of 'dummy_test.go' for dummy test files")
	flag.Parse()

	innerCommand := flag.Args()

	testPackages, isSet := os.LookupEnv("GO_TEST_PACKAGES")
	if isSet {
		inputPackages = testPackages
	}

	packages = getAllPackages(inputPackages)

	log.Printf("packages=%s\n", packages)

	createdDummyTests := make([]string, 0)
	for _, p := range packages {
		if len(p.TestGoFiles) == 0 {
			testName := "dummy_test.go"
			if usePackageTestName {
				testName = fmt.Sprintf("%s_test.go", p.Name)
			}
			dummyTestPath := filepath.Join(p.Dir, testName)
			err := os.WriteFile(dummyTestPath, []byte(fmt.Sprintf("package %s", p.Name)), 0644)
			if err != nil {
				log.Printf("Cannot create dummy test file on path: %s. Cause: %s\n", dummyTestPath, err.Error())
			} else {
				createdDummyTests = append(createdDummyTests, dummyTestPath)
			}
		}
	}
	log.Printf("Created dummy tests: %v\n", createdDummyTests)

	if len(innerCommand) > 0 {
		out, err := exec.Command(innerCommand[0], innerCommand[1:]...).CombinedOutput()
		if err != nil {
			cleanCreatedData(createdDummyTests)
			log.Fatalf("Error occured: %s, %s\n", string(out), err.Error())
		}
		_, err = os.Stdin.Write(out)
		if err != nil {
			cleanCreatedData(createdDummyTests)
			log.Fatalf("Error during writing to stdin: %s\n", err.Error())
		}
	}

	if cleanup {
		cleanCreatedData(createdDummyTests)
	}
	fmt.Println("All done!")
}

func getFormattedPackages(packages string) []string {
	input := strings.TrimSpace(packages)
	return strings.Split(regexp.MustCompile(`\s`).ReplaceAllString(input, " "), " ")
}

func getAllPackages(pkgs string) []Package {
	command := []string{"list", "-json"}
	command = append(command, getFormattedPackages(pkgs)...)
	out, err := exec.Command("go", command...).CombinedOutput()
	if err != nil {
		log.Fatalf("Error during packages acquiring: %s, %s", string(out), err.Error())
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

func cleanCreatedData(createdFiles []string) {
	for _, dt := range createdFiles {
		err := os.Remove(dt)
		if err != nil {
			log.Printf("Cleanup failed for %s", dt)
		}
	}
}