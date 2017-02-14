// +build BPMS

package BPMS

type FlowNodeInstance interface {
	/** Method of FlowNodeInstance **/

	/** UUID **/
	GetUUID() string

	/** FlowNodeType **/
	GetFlowNodeType() FlowNodeType
	SetFlowNodeType(interface{})

	/** LifecycleState **/
	GetLifecycleState() LifecycleState
	SetLifecycleState(interface{})

	/** InputRef **/
	GetInputRef() []*ConnectingObject
	SetInputRef(interface{})

	/** OutputRef **/
	GetOutputRef() []*ConnectingObject
	SetOutputRef(interface{})

	/** Process instance ptr **/
	GetProcessInstancePtr() *ProcessInstance
}
