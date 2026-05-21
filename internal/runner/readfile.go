package runner

import "os"

// readFileNative is a cross-platform readback used by tests so they
// work on Windows where `cat` is not available.
func readFileNative(p string) ([]byte, error) {
	return os.ReadFile(p)
}
