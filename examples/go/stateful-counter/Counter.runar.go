package contract

import runar "github.com/icellan/runar/packages/runar-go"

type Counter struct {
	runar.StatefulSmartContract
	Count runar.Bigint // no tag = mutable (stateful)
}

func (c *Counter) Increment() {
	c.Count++
}

func (c *Counter) Decrement() {
	runar.Assert(c.Count > 0)
	c.Count--
}
