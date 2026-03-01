package tsop

import (
	"fmt"
	"os"
	"strings"

	"github.com/tsop/compiler-go/frontend"
)

// CompileCheck runs the TSOP frontend (parse → validate → typecheck) on a
// .tsop.go contract file. Returns nil if the contract is valid TSOP.
//
// Use this in tests alongside business logic tests to ensure the contract
// will compile to Bitcoin Script:
//
//	func TestCompile(t *testing.T) {
//	    if err := tsop.CompileCheck("MyContract.tsop.go"); err != nil {
//	        t.Fatalf("TSOP compile check failed: %v", err)
//	    }
//	}
func CompileCheck(contractFile string) error {
	source, err := os.ReadFile(contractFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", contractFile, err)
	}

	result := frontend.ParseSource(source, contractFile)
	if len(result.Errors) > 0 {
		return fmt.Errorf("parse errors: %s", strings.Join(result.Errors, "; "))
	}
	if result.Contract == nil {
		return fmt.Errorf("no contract found in %s", contractFile)
	}

	v := frontend.Validate(result.Contract)
	if len(v.Errors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(v.Errors, "; "))
	}

	tc := frontend.TypeCheck(result.Contract)
	if len(tc.Errors) > 0 {
		return fmt.Errorf("type check errors: %s", strings.Join(tc.Errors, "; "))
	}

	return nil
}
