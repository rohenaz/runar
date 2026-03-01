package contract

import "tsop"

type OraclePriceFeed struct {
	tsop.SmartContract
	OraclePubKey tsop.RabinPubKey `tsop:"readonly"`
	Receiver     tsop.PubKey      `tsop:"readonly"`
}

func (c *OraclePriceFeed) Settle(price tsop.Bigint, rabinSig tsop.RabinSig, padding tsop.ByteString, sig tsop.Sig) {
	// Verify oracle signed this price
	msg := tsop.Num2Bin(price, 8)

	tsop.Assert(tsop.VerifyRabinSig(msg, rabinSig, padding, c.OraclePubKey))

	// Price must be above threshold for payout
	tsop.Assert(price > 50000)

	// Receiver must sign
	tsop.Assert(tsop.CheckSig(sig, c.Receiver))
}
