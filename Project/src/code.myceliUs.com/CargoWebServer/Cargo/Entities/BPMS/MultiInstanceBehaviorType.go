// +build BPMS

package BPMS

type MultiInstanceBehaviorType int
const(
	MultiInstanceBehaviorType_None MultiInstanceBehaviorType = 1+iota
	MultiInstanceBehaviorType_One
	MultiInstanceBehaviorType_All
	MultiInstanceBehaviorType_Complex
)
