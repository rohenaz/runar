package contract

import "tsop"

type Stateful struct {
	tsop.StatefulSmartContract
	Count    tsop.Int
	MaxCount tsop.Int `tsop:"readonly"`
}

func (c *Stateful) Increment(amount tsop.Int, txPreimage tsop.SigHashPreimage) {
	tsop.Assert(tsop.CheckPreimage(txPreimage))
	c.Count = c.Count + amount
	tsop.Assert(c.Count <= c.MaxCount)
	tsop.Assert(tsop.Hash256(c.GetStateScript()) == tsop.ExtractOutputHash(txPreimage))
}

func (c *Stateful) Reset(txPreimage tsop.SigHashPreimage) {
	tsop.Assert(tsop.CheckPreimage(txPreimage))
	c.Count = 0
	tsop.Assert(tsop.Hash256(c.GetStateScript()) == tsop.ExtractOutputHash(txPreimage))
}
