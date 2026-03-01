package contract

import "tsop"

type Arithmetic struct {
	tsop.SmartContract
	Target tsop.Int `tsop:"readonly"`
}

func (c *Arithmetic) Verify(a tsop.Int, b tsop.Int) {
	sum := a + b
	diff := a - b
	prod := a * b
	quot := a / b
	result := sum + diff + prod + quot
	tsop.Assert(result == c.Target)
}
