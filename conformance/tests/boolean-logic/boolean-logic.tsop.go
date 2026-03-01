package contract

import "tsop"

type BooleanLogic struct {
	tsop.SmartContract
	Threshold tsop.Int `tsop:"readonly"`
}

func (c *BooleanLogic) Verify(a tsop.Int, b tsop.Int, flag tsop.Bool) {
	aAboveThreshold := a > c.Threshold
	bAboveThreshold := b > c.Threshold
	bothAbove := aAboveThreshold && bAboveThreshold
	eitherAbove := aAboveThreshold || bAboveThreshold
	notFlag := !flag
	tsop.Assert(bothAbove || (eitherAbove && notFlag))
}
