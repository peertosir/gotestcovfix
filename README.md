#### GoTestCovFix

Go test overall coverage can be calculated in some weird way.

If you are using Golang >= 1.20 and some other versions, packages without test files will not be included in overall coverage.

This package acts like a wrapper for any test launch cli you are using(go test, gotestsum or any other).

It will create for all packages you chose, and can clear them after test launch.


Args:
- **-tpkgs "<values...>"** : list of packages that should be included into overall coverage calculation, separated by space. Default value: "./...". This value can also be set with env. variable **GO_TEST_PACKAGES** and env. variable has **higher** priority
- **-cleanup** : delete all dummy test files after test launch. Can be useful in local launches(if you dont want to leave them in repo)
- **-pkgnames** : dummy files with test will have name {package}_test.go, instead of dummy_test.go


Example of usage:
```
gotestcovfix -tpkgs "gotestcov/pkg/utils gotestcov/internal/db" -cleanup -pkgnames go test -cover -coverprofile=coverage.tmp -coverpkg=./...
```

You can also pass result of ```go list ./...``` as -tpkgs arg value, especially if you have some exclude logic in your launch scripts.
