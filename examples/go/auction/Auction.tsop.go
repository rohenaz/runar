package contract

import "tsop"

type Auction struct {
	tsop.StatefulSmartContract
	Auctioneer    tsop.PubKey `tsop:"readonly"`
	HighestBidder tsop.PubKey // stateful
	HighestBid    tsop.Bigint // stateful
	Deadline      tsop.Bigint `tsop:"readonly"` // block height deadline
}

// State-mutating: compiler auto-injects checkPreimage + state continuation
func (c *Auction) Bid(bidder tsop.PubKey, bidAmount tsop.Bigint) {
	// Bid must be higher than current highest
	tsop.Assert(bidAmount > c.HighestBid)

	// Auction must not have ended
	tsop.Assert(tsop.ExtractLocktime(c.TxPreimage) < c.Deadline)

	// Update state
	c.HighestBidder = bidder
	c.HighestBid = bidAmount
}

// Non-mutating: compiler auto-injects checkPreimage only (no state continuation)
func (c *Auction) Close(sig tsop.Sig) {
	// Only auctioneer can close
	tsop.Assert(tsop.CheckSig(sig, c.Auctioneer))

	// Auction must have ended
	tsop.Assert(tsop.ExtractLocktime(c.TxPreimage) >= c.Deadline)
}
