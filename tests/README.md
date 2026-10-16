# Why the tests are not in this folder

Go has no test directory. **A directory is a package**, so a test file in
`tests/` belongs to a *different* package than the code it tests and can only
see exported identifiers. Verified with a scratch module:

```
# exp/tests [exp/tests.test]
tests\lib_test.go:16:10: name secret not exported by package pkg
```

Even giving the test file the same package name does not help — the package is
identified by its directory, not its name:

```
# exp2/tests [exp2/tests.test]
tests\lib_test.go:6:5: undefined: helper
```

The tests here use **31 unexported identifiers** (`buildSidebarRows`,
`sessionGroup`, `fittedPanelWidths`, `mask`, `humanTokens`, `kerenzikovTheme`,
`revealBatch`, `rowHeader`, `sidebarWidthDefault`, …). Moving them would mean
exporting all of them purely for tests, which leaks the app's internals into a
public API for no benefit.

So tests live beside the code they test, as Go's own convention does:

```
src/app.go            src/main_test.go
src/settings.go       src/settings_test.go
src/layout.go         src/layout_test.go
src/ui_test.go        shared pure-logic helpers and their tests
internal/daemon/daemon.go    internal/daemon/daemon_test.go
internal/opencode/service.go internal/opencode/service_test.go
```

`go test ./...` runs them all. `go tool mygo vet .` adds MyGo's native UI
build-lifetime checks.
