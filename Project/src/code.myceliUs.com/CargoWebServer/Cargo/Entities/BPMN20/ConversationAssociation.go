package BPMN20

import(
"encoding/xml"
)

type ConversationAssociation struct{

	/** The entity UUID **/
	UUID string
	/** The entity TypeName **/
	TYPENAME string
	/** If the entity value has change... **/
	NeedSave bool

	/** If the entity is fully initialyse **/
	IsInit   bool

	/** members of BaseElement **/
	M_id string
	m_other interface{}
	/** If the ref is a string and not an object **/
	M_other string
	M_extensionElements *ExtensionElements
	M_extensionDefinitions []*ExtensionDefinition
	M_extensionValues []*ExtensionAttributeValue
	M_documentation []*Documentation

	/** members of ConversationAssociation **/
	m_innerConversationNodeRef ConversationNode
	/** If the ref is a string and not an object **/
	M_innerConversationNodeRef string
	m_outerConversationNodeRef ConversationNode
	/** If the ref is a string and not an object **/
	M_outerConversationNodeRef string


	/** Associations **/
	m_converstaionAssociationsPtr Collaboration
	/** If the ref is a string and not an object **/
	M_converstaionAssociationsPtr string
	m_lanePtr []*Lane
	/** If the ref is a string and not an object **/
	M_lanePtr []string
	m_outgoingPtr []*Association
	/** If the ref is a string and not an object **/
	M_outgoingPtr []string
	m_incomingPtr []*Association
	/** If the ref is a string and not an object **/
	M_incomingPtr []string
}

/** Xml parser for ConversationAssociation **/
type XsdConversationAssociation struct {
	XMLName xml.Name	`xml:"conversationAssociation"`
	/** BaseElement **/
	M_documentation	[]*XsdDocumentation	`xml:"documentation,omitempty"`
	M_extensionElements	*XsdExtensionElements	`xml:"extensionElements,omitempty"`
	M_id	string	`xml:"id,attr"`
//	M_other	string	`xml:",innerxml"`


	M_innerConversationNodeRef	string	`xml:"innerConversationNodeRef,attr"`
	M_outerConversationNodeRef	string	`xml:"outerConversationNodeRef,attr"`

}
/** UUID **/
func (this *ConversationAssociation) GetUUID() string{
	return this.UUID
}

/** Id **/
func (this *ConversationAssociation) GetId() string{
	return this.M_id
}

/** Init reference Id **/
func (this *ConversationAssociation) SetId(ref interface{}){
	this.NeedSave = true
	this.M_id = ref.(string)
}

/** Remove reference Id **/

/** Other **/
func (this *ConversationAssociation) GetOther() interface{}{
	return this.M_other
}

/** Init reference Other **/
func (this *ConversationAssociation) SetOther(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_other = ref.(string)
	}else{
		this.m_other = ref.(interface{})
	}
}

/** Remove reference Other **/

/** ExtensionElements **/
func (this *ConversationAssociation) GetExtensionElements() *ExtensionElements{
	return this.M_extensionElements
}

/** Init reference ExtensionElements **/
func (this *ConversationAssociation) SetExtensionElements(ref interface{}){
	this.NeedSave = true
	this.M_extensionElements = ref.(*ExtensionElements)
}

/** Remove reference ExtensionElements **/

/** ExtensionDefinitions **/
func (this *ConversationAssociation) GetExtensionDefinitions() []*ExtensionDefinition{
	return this.M_extensionDefinitions
}

/** Init reference ExtensionDefinitions **/
func (this *ConversationAssociation) SetExtensionDefinitions(ref interface{}){
	this.NeedSave = true
	isExist := false
	var extensionDefinitionss []*ExtensionDefinition
	for i:=0; i<len(this.M_extensionDefinitions); i++ {
		if this.M_extensionDefinitions[i].GetName() != ref.(*ExtensionDefinition).GetName() {
			extensionDefinitionss = append(extensionDefinitionss, this.M_extensionDefinitions[i])
		} else {
			isExist = true
			extensionDefinitionss = append(extensionDefinitionss, ref.(*ExtensionDefinition))
		}
	}
	if !isExist {
		extensionDefinitionss = append(extensionDefinitionss, ref.(*ExtensionDefinition))
	}
	this.M_extensionDefinitions = extensionDefinitionss
}

/** Remove reference ExtensionDefinitions **/

/** ExtensionValues **/
func (this *ConversationAssociation) GetExtensionValues() []*ExtensionAttributeValue{
	return this.M_extensionValues
}

/** Init reference ExtensionValues **/
func (this *ConversationAssociation) SetExtensionValues(ref interface{}){
	this.NeedSave = true
	isExist := false
	var extensionValuess []*ExtensionAttributeValue
	for i:=0; i<len(this.M_extensionValues); i++ {
		if this.M_extensionValues[i].GetUUID() != ref.(*ExtensionAttributeValue).GetUUID() {
			extensionValuess = append(extensionValuess, this.M_extensionValues[i])
		} else {
			isExist = true
			extensionValuess = append(extensionValuess, ref.(*ExtensionAttributeValue))
		}
	}
	if !isExist {
		extensionValuess = append(extensionValuess, ref.(*ExtensionAttributeValue))
	}
	this.M_extensionValues = extensionValuess
}

/** Remove reference ExtensionValues **/

/** Documentation **/
func (this *ConversationAssociation) GetDocumentation() []*Documentation{
	return this.M_documentation
}

/** Init reference Documentation **/
func (this *ConversationAssociation) SetDocumentation(ref interface{}){
	this.NeedSave = true
	isExist := false
	var documentations []*Documentation
	for i:=0; i<len(this.M_documentation); i++ {
		if this.M_documentation[i].GetUUID() != ref.(BaseElement).GetUUID() {
			documentations = append(documentations, this.M_documentation[i])
		} else {
			isExist = true
			documentations = append(documentations, ref.(*Documentation))
		}
	}
	if !isExist {
		documentations = append(documentations, ref.(*Documentation))
	}
	this.M_documentation = documentations
}

/** Remove reference Documentation **/
func (this *ConversationAssociation) RemoveDocumentation(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	documentation_ := make([]*Documentation, 0)
	for i := 0; i < len(this.M_documentation); i++ {
		if toDelete.GetUUID() != this.M_documentation[i].GetUUID() {
			documentation_ = append(documentation_, this.M_documentation[i])
		}
	}
	this.M_documentation = documentation_
}

/** InnerConversationNodeRef **/
func (this *ConversationAssociation) GetInnerConversationNodeRef() ConversationNode{
	return this.m_innerConversationNodeRef
}

/** Init reference InnerConversationNodeRef **/
func (this *ConversationAssociation) SetInnerConversationNodeRef(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_innerConversationNodeRef = ref.(string)
	}else{
		this.m_innerConversationNodeRef = ref.(ConversationNode)
		this.M_innerConversationNodeRef = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference InnerConversationNodeRef **/
func (this *ConversationAssociation) RemoveInnerConversationNodeRef(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_innerConversationNodeRef.(BaseElement).GetUUID() {
		this.m_innerConversationNodeRef = nil
		this.M_innerConversationNodeRef = ""
	}
}

/** OuterConversationNodeRef **/
func (this *ConversationAssociation) GetOuterConversationNodeRef() ConversationNode{
	return this.m_outerConversationNodeRef
}

/** Init reference OuterConversationNodeRef **/
func (this *ConversationAssociation) SetOuterConversationNodeRef(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_outerConversationNodeRef = ref.(string)
	}else{
		this.m_outerConversationNodeRef = ref.(ConversationNode)
		this.M_outerConversationNodeRef = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference OuterConversationNodeRef **/
func (this *ConversationAssociation) RemoveOuterConversationNodeRef(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_outerConversationNodeRef.(BaseElement).GetUUID() {
		this.m_outerConversationNodeRef = nil
		this.M_outerConversationNodeRef = ""
	}
}

/** ConverstaionAssociations **/
func (this *ConversationAssociation) GetConverstaionAssociationsPtr() Collaboration{
	return this.m_converstaionAssociationsPtr
}

/** Init reference ConverstaionAssociations **/
func (this *ConversationAssociation) SetConverstaionAssociationsPtr(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_converstaionAssociationsPtr = ref.(string)
	}else{
		this.m_converstaionAssociationsPtr = ref.(Collaboration)
		this.M_converstaionAssociationsPtr = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference ConverstaionAssociations **/
func (this *ConversationAssociation) RemoveConverstaionAssociationsPtr(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_converstaionAssociationsPtr.(BaseElement).GetUUID() {
		this.m_converstaionAssociationsPtr = nil
		this.M_converstaionAssociationsPtr = ""
	}
}

/** Lane **/
func (this *ConversationAssociation) GetLanePtr() []*Lane{
	return this.m_lanePtr
}

/** Init reference Lane **/
func (this *ConversationAssociation) SetLanePtr(ref interface{}){
	this.NeedSave = true
	if refStr, ok := ref.(string); ok {
		for i:=0; i < len(this.M_lanePtr); i++ {
			if this.M_lanePtr[i] == refStr {
				return
			}
		}
		this.M_lanePtr = append(this.M_lanePtr, ref.(string))
	}else{
		this.RemoveLanePtr(ref)
		this.m_lanePtr = append(this.m_lanePtr, ref.(*Lane))
		this.M_lanePtr = append(this.M_lanePtr, ref.(BaseElement).GetUUID())
	}
}

/** Remove reference Lane **/
func (this *ConversationAssociation) RemoveLanePtr(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	lanePtr_ := make([]*Lane, 0)
	lanePtrUuid := make([]string, 0)
	for i := 0; i < len(this.m_lanePtr); i++ {
		if toDelete.GetUUID() != this.m_lanePtr[i].GetUUID() {
			lanePtr_ = append(lanePtr_, this.m_lanePtr[i])
			lanePtrUuid = append(lanePtrUuid, this.M_lanePtr[i])
		}
	}
	this.m_lanePtr = lanePtr_
	this.M_lanePtr = lanePtrUuid
}

/** Outgoing **/
func (this *ConversationAssociation) GetOutgoingPtr() []*Association{
	return this.m_outgoingPtr
}

/** Init reference Outgoing **/
func (this *ConversationAssociation) SetOutgoingPtr(ref interface{}){
	this.NeedSave = true
	if refStr, ok := ref.(string); ok {
		for i:=0; i < len(this.M_outgoingPtr); i++ {
			if this.M_outgoingPtr[i] == refStr {
				return
			}
		}
		this.M_outgoingPtr = append(this.M_outgoingPtr, ref.(string))
	}else{
		this.RemoveOutgoingPtr(ref)
		this.m_outgoingPtr = append(this.m_outgoingPtr, ref.(*Association))
		this.M_outgoingPtr = append(this.M_outgoingPtr, ref.(BaseElement).GetUUID())
	}
}

/** Remove reference Outgoing **/
func (this *ConversationAssociation) RemoveOutgoingPtr(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	outgoingPtr_ := make([]*Association, 0)
	outgoingPtrUuid := make([]string, 0)
	for i := 0; i < len(this.m_outgoingPtr); i++ {
		if toDelete.GetUUID() != this.m_outgoingPtr[i].GetUUID() {
			outgoingPtr_ = append(outgoingPtr_, this.m_outgoingPtr[i])
			outgoingPtrUuid = append(outgoingPtrUuid, this.M_outgoingPtr[i])
		}
	}
	this.m_outgoingPtr = outgoingPtr_
	this.M_outgoingPtr = outgoingPtrUuid
}

/** Incoming **/
func (this *ConversationAssociation) GetIncomingPtr() []*Association{
	return this.m_incomingPtr
}

/** Init reference Incoming **/
func (this *ConversationAssociation) SetIncomingPtr(ref interface{}){
	this.NeedSave = true
	if refStr, ok := ref.(string); ok {
		for i:=0; i < len(this.M_incomingPtr); i++ {
			if this.M_incomingPtr[i] == refStr {
				return
			}
		}
		this.M_incomingPtr = append(this.M_incomingPtr, ref.(string))
	}else{
		this.RemoveIncomingPtr(ref)
		this.m_incomingPtr = append(this.m_incomingPtr, ref.(*Association))
		this.M_incomingPtr = append(this.M_incomingPtr, ref.(BaseElement).GetUUID())
	}
}

/** Remove reference Incoming **/
func (this *ConversationAssociation) RemoveIncomingPtr(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	incomingPtr_ := make([]*Association, 0)
	incomingPtrUuid := make([]string, 0)
	for i := 0; i < len(this.m_incomingPtr); i++ {
		if toDelete.GetUUID() != this.m_incomingPtr[i].GetUUID() {
			incomingPtr_ = append(incomingPtr_, this.m_incomingPtr[i])
			incomingPtrUuid = append(incomingPtrUuid, this.M_incomingPtr[i])
		}
	}
	this.m_incomingPtr = incomingPtr_
	this.M_incomingPtr = incomingPtrUuid
}