// +build integration

package main

import (
	"os"
	"strings"
	"testing"

	"boscoin.io/sebak/cmd/sebak/cmd"
)

// Run the program as a test
// This needs to be compiled with `go test` with special flags (see tests/run.sh)
// to do the trick.
// It filters out test arguments for the main, then launch the mail.
// This allows us to gather coverage reports from integration tests
func TestIntegration(t *testing.T) {
	var filteredArgs []string
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-test.") ||
			strings.HasPrefix(arg, "-httptest.") {
			continue
		}
		filteredArgs = append(filteredArgs, arg)
	}
	cmd.SetArgs(filteredArgs)
	main()
}
