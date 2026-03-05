//go:build integration

package integration

import (
	"fmt"
	"os"
	"testing"

	"runar-integration/helpers"
)

func TestMain(m *testing.M) {
	if !helpers.IsNodeAvailable() {
		fmt.Fprintln(os.Stderr, "Regtest node not running. Skipping integration tests.")
		fmt.Fprintln(os.Stderr, "Start with: cd integration && ./regtest.sh start")
		os.Exit(0)
	}

	// Mine initial blocks so the wallet has coins
	if err := helpers.Mine(101); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to mine initial blocks: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
