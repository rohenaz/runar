package contract

import (
	"testing"
	"tsop"
)

func newNFT(owner tsop.PubKey) *SimpleNFT {
	return &SimpleNFT{
		Owner:    owner,
		TokenId:  tsop.ByteString("unique-nft-001"),
		Metadata: tsop.ByteString("ipfs://QmTest"),
	}
}

func TestSimpleNFT_Transfer(t *testing.T) {
	alice := tsop.PubKey("alice_pubkey_33bytes_placeholder!")
	bob := tsop.PubKey("bob___pubkey_33bytes_placeholder!")
	c := newNFT(alice)
	c.Transfer(tsop.MockSig(), bob, 1000)
	if len(c.Outputs()) != 1 {
		t.Fatalf("expected 1 output, got %d", len(c.Outputs()))
	}
}

func TestSimpleNFT_Burn(t *testing.T) {
	alice := tsop.PubKey("alice_pubkey_33bytes_placeholder!")
	c := newNFT(alice)
	c.Burn(tsop.MockSig())
	if len(c.Outputs()) != 0 {
		t.Errorf("expected 0 outputs after burn, got %d", len(c.Outputs()))
	}
}

func TestSimpleNFT_Compile(t *testing.T) {
	if err := tsop.CompileCheck("NFTExample.tsop.go"); err != nil {
		t.Fatalf("TSOP compile check failed: %v", err)
	}
}
