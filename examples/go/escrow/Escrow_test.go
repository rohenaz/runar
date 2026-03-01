package contract

import (
	"testing"
	"tsop"
)

func newEscrow() *Escrow {
	return &Escrow{
		Buyer:   tsop.MockPubKey(),
		Seller:  tsop.MockPubKey(),
		Arbiter: tsop.MockPubKey(),
	}
}

func TestEscrow_ReleaseBySeller(t *testing.T)  { newEscrow().ReleaseBySeller(tsop.MockSig()) }
func TestEscrow_ReleaseByArbiter(t *testing.T) { newEscrow().ReleaseByArbiter(tsop.MockSig()) }
func TestEscrow_RefundToBuyer(t *testing.T)    { newEscrow().RefundToBuyer(tsop.MockSig()) }
func TestEscrow_RefundByArbiter(t *testing.T)  { newEscrow().RefundByArbiter(tsop.MockSig()) }

func TestEscrow_Compile(t *testing.T) {
	if err := tsop.CompileCheck("Escrow.tsop.go"); err != nil {
		t.Fatalf("TSOP compile check failed: %v", err)
	}
}
