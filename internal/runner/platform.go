package runner

import "runtime"

// runtimeIsWindows returns true on Windows builds. Extracted into a
// helper so callers can test platform-specific paths without an
// import of the runtime package leaking into runner.go's review
// diffs.
func runtimeIsWindows() bool { return runtime.GOOS == "windows" }
