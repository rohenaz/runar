package contract

import "tsop"

type BoundedLoop struct {
	tsop.SmartContract
	ExpectedSum tsop.Int `tsop:"readonly"`
}

func (c *BoundedLoop) Verify(start tsop.Int) {
	sum := tsop.Int(0)
	for i := tsop.Int(0); i < 5; i++ {
		sum = sum + start + i
	}
	tsop.Assert(sum == c.ExpectedSum)
}
