package contract

import (
	"testing"
	"tsop"
)

func newOracleFeed() *OraclePriceFeed {
	return &OraclePriceFeed{
		OraclePubKey: tsop.RabinPubKey("oracle_rabin_pk"),
		Receiver:     tsop.MockPubKey(),
	}
}

func TestOraclePriceFeed_Settle(t *testing.T) {
	newOracleFeed().Settle(60000, tsop.RabinSig("sig"), tsop.ByteString("pad"), tsop.MockSig())
}

func TestOraclePriceFeed_Settle_PriceTooLow_Fails(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected assertion failure")
		}
	}()
	newOracleFeed().Settle(50000, tsop.RabinSig("sig"), tsop.ByteString("pad"), tsop.MockSig())
}

func TestOraclePriceFeed_Compile(t *testing.T) {
	if err := tsop.CompileCheck("OraclePriceFeed.tsop.go"); err != nil {
		t.Fatalf("TSOP compile check failed: %v", err)
	}
}
