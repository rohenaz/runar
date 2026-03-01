package contract

import "tsop"

type Escrow struct {
	tsop.SmartContract
	Buyer  tsop.PubKey `tsop:"readonly"`
	Seller tsop.PubKey `tsop:"readonly"`
	Arbiter tsop.PubKey `tsop:"readonly"`
}

func (c *Escrow) ReleaseBySeller(sig tsop.Sig) {
	tsop.Assert(tsop.CheckSig(sig, c.Seller))
}

func (c *Escrow) ReleaseByArbiter(sig tsop.Sig) {
	tsop.Assert(tsop.CheckSig(sig, c.Arbiter))
}

func (c *Escrow) RefundToBuyer(sig tsop.Sig) {
	tsop.Assert(tsop.CheckSig(sig, c.Buyer))
}

func (c *Escrow) RefundByArbiter(sig tsop.Sig) {
	tsop.Assert(tsop.CheckSig(sig, c.Arbiter))
}
