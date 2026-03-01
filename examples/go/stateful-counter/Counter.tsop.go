package contract

import "tsop"

type Counter struct {
	tsop.StatefulSmartContract
	Count tsop.Bigint // no tag = mutable (stateful)
}

func (c *Counter) Increment() {
	c.Count++
}

func (c *Counter) Decrement() {
	tsop.Assert(c.Count > 0)
	c.Count--
}
