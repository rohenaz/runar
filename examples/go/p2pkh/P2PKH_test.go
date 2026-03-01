package contract

import (
	"testing"

	"tsop"
)

func TestP2PKH_Unlock(t *testing.T) {
	pk := tsop.MockPubKey()
	c := &P2PKH{PubKeyHash: tsop.Hash160(pk)}
	c.Unlock(tsop.MockSig(), pk)
}

func TestP2PKH_Unlock_WrongKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected assertion failure for wrong public key")
		}
	}()
	pk := tsop.MockPubKey()
	wrongPk := tsop.PubKey("\x03" + string(make([]byte, 32)))
	c := &P2PKH{PubKeyHash: tsop.Hash160(pk)}
	c.Unlock(tsop.MockSig(), wrongPk)
}

func TestP2PKH_Compile(t *testing.T) {
	if err := tsop.CompileCheck("P2PKH.tsop.go"); err != nil {
		t.Fatalf("TSOP compile check failed: %v", err)
	}
}
