package contract

import (
	"testing"
	"tsop"
)

func newVault() *CovenantVault {
	return &CovenantVault{
		Owner:     tsop.MockPubKey(),
		Recipient: tsop.Hash160(tsop.MockPubKey()),
		MinAmount: 1000,
	}
}

func TestCovenantVault_Spend(t *testing.T) {
	newVault().Spend(tsop.MockSig(), 5000, tsop.MockPreimage())
}

func TestCovenantVault_Spend_ExactMinimum(t *testing.T) {
	newVault().Spend(tsop.MockSig(), 1000, tsop.MockPreimage())
}

func TestCovenantVault_Spend_BelowMinimum_Fails(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected assertion failure")
		}
	}()
	newVault().Spend(tsop.MockSig(), 999, tsop.MockPreimage())
}

func TestCovenantVault_Compile(t *testing.T) {
	if err := tsop.CompileCheck("CovenantVault.tsop.go"); err != nil {
		t.Fatalf("TSOP compile check failed: %v", err)
	}
}
