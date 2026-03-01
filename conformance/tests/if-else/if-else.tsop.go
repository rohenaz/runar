package contract

import "tsop"

type IfElse struct {
	tsop.SmartContract
	Limit tsop.Int `tsop:"readonly"`
}

func (c *IfElse) Check(value tsop.Int, mode tsop.Bool) {
	result := tsop.Int(0)
	if mode {
		result = value + c.Limit
	} else {
		result = value - c.Limit
	}
	tsop.Assert(result > 0)
}
