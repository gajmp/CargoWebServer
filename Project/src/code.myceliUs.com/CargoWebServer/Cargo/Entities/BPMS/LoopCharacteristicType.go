// +build BPMS

package BPMS

type LoopCharacteristicType int
const(
	LoopCharacteristicType_StandardLoopCharacteristics LoopCharacteristicType = 1+iota
	LoopCharacteristicType_MultiInstanceLoopCharacteristics
)
