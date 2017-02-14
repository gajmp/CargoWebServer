// +build BPMN20

package BPMN20

import(
"encoding/xml"
)

type Escalation struct{

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

	/** members of RootElement **/
	/** No members **/

	/** members of Escalation **/
	m_structureRef *ItemDefinition
	/** If the ref is a string and not an object **/
	M_structureRef string
	M_name string
	M_escalationCode string


	/** Associations **/
	m_escalationEventDefinitionPtr []*EscalationEventDefinition
	/** If the ref is a string and not an object **/
	M_escalationEventDefinitionPtr []string
	m_lanePtr []*Lane
	/** If the ref is a string and not an object **/
	M_lanePtr []string
	m_outgoingPtr []*Association
	/** If the ref is a string and not an object **/
	M_outgoingPtr []string
	m_incomingPtr []*Association
	/** If the ref is a string and not an object **/
	M_incomingPtr []string
	m_definitionsPtr *Definitions
	/** If the ref is a string and not an object **/
	M_definitionsPtr string
}

/** Xml parser for Escalation **/
type XsdEscalation struct {
	XMLName xml.Name	`xml:"escalation"`
	/** BaseElement **/
	M_documentation	[]*XsdDocumentation	`xml:"documentation,omitempty"`
	M_extensionElements	*XsdExtensionElements	`xml:"extensionElements,omitempty"`
	M_id	string	`xml:"id,attr"`
//	M_other	string	`xml:",innerxml"`


	/** RootElement **/


	M_name	string	`xml:"name,attr"`
	M_escalationCode	string	`xml:"escalationCode,attr"`
	M_structureRef	string	`xml:"structureRef,attr"`

}
/** UUID **/
func (this *Escalation) GetUUID() string{
	return this.UUID
}

/** Id **/
func (this *Escalation) GetId() string{
	return this.M_id
}

/** Init reference Id **/
func (this *Escalation) SetId(ref interface{}){
	this.NeedSave = true
	this.M_id = ref.(string)
}

/** Remove reference Id **/

/** Other **/
func (this *Escalation) GetOther() interface{}{
	return this.M_other
}

/** Init reference Other **/
func (this *Escalation) SetOther(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_other = ref.(string)
	}else{
		this.m_other = ref.(interface{})
	}
}

/** Remove reference Other **/

/** ExtensionElements **/
func (this *Escalation) GetExtensionElements() *ExtensionElements{
	return this.M_extensionElements
}

/** Init reference ExtensionElements **/
func (this *Escalation) SetExtensionElements(ref interface{}){
	this.NeedSave = true
	this.M_extensionElements = ref.(*ExtensionElements)
}

/** Remove reference ExtensionElements **/
func (this *Escalation) RemoveExtensionElements(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(*ExtensionElements)
	if toDelete.GetUUID() == this.M_extensionElements.GetUUID() {
		this.M_extensionElements = nil
	}
}

/** ExtensionDefinitions **/
func (this *Escalation) GetExtensionDefinitions() []*ExtensionDefinition{
	return this.M_extensionDefinitions
}

/** Init reference ExtensionDefinitions **/
func (this *Escalation) SetExtensionDefinitions(ref interface{}){
	this.NeedSave = true
	isExist := false
	var extensionDefinitionss []*ExtensionDefinition
	for i:=0; i<len(this.M_extensionDefinitions); i++ {
		if this.M_extensionDefinitions[i].GetUUID() != ref.(*ExtensionDefinition).GetUUID() {
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
func (this *Escalation) RemoveExtensionDefinitions(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(*ExtensionDefinition)
	extensionDefinitions_ := make([]*ExtensionDefinition, 0)
	for i := 0; i < len(this.M_extensionDefinitions); i++ {
		if toDelete.GetUUID() != this.M_extensionDefinitions[i].GetUUID() {
			extensionDefinitions_ = append(extensionDefinitions_, this.M_extensionDefinitions[i])
		}
	}
	this.M_extensionDefinitions = extensionDefinitions_
}

/** ExtensionValues **/
func (this *Escalation) GetExtensionValues() []*ExtensionAttributeValue{
	return this.M_extensionValues
}

/** Init reference ExtensionValues **/
func (this *Escalation) SetExtensionValues(ref interface{}){
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
func (this *Escalation) RemoveExtensionValues(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(*ExtensionAttributeValue)
	extensionValues_ := make([]*ExtensionAttributeValue, 0)
	for i := 0; i < len(this.M_extensionValues); i++ {
		if toDelete.GetUUID() != this.M_extensionValues[i].GetUUID() {
			extensionValues_ = append(extensionValues_, this.M_extensionValues[i])
		}
	}
	this.M_extensionValues = extensionValues_
}

/** Documentation **/
func (this *Escalation) GetDocumentation() []*Documentation{
	return this.M_documentation
}

/** Init reference Documentation **/
func (this *Escalation) SetDocumentation(ref interface{}){
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
func (this *Escalation) RemoveDocumentation(ref interface{}){
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

/** StructureRef **/
func (this *Escalation) GetStructureRef() *ItemDefinition{
	return this.m_structureRef
}

/** Init reference StructureRef **/
func (this *Escalation) SetStructureRef(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_structureRef = ref.(string)
	}else{
		this.m_structureRef = ref.(*ItemDefinition)
		this.M_structureRef = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference StructureRef **/
func (this *Escalation) RemoveStructureRef(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_structureRef.GetUUID() {
		this.m_structureRef = nil
		this.M_structureRef = ""
	}
}

/** Name **/
func (this *Escalation) GetName() string{
	return this.M_name
}

/** Init reference Name **/
func (this *Escalation) SetName(ref interface{}){
	this.NeedSave = true
	this.M_name = ref.(string)
}

/** Remove reference Name **/

/** EscalationCode **/
func (this *Escalation) GetEscalationCode() string{
	return this.M_escalationCode
}

/** Init reference EscalationCode **/
func (this *Escalation) SetEscalationCode(ref interface{}){
	this.NeedSave = true
	this.M_escalationCode = ref.(string)
}

/** Remove reference EscalationCode **/

/** EscalationEventDefinition **/
func (this *Escalation) GetEscalationEventDefinitionPtr() []*EscalationEventDefinition{
	return this.m_escalationEventDefinitionPtr
}

/** Init reference EscalationEventDefinition **/
func (this *Escalation) SetEscalationEventDefinitionPtr(ref interface{}){
	this.NeedSave = true
	if refStr, ok := ref.(string); ok {
		for i:=0; i < len(this.M_escalationEventDefinitionPtr); i++ {
			if this.M_escalationEventDefinitionPtr[i] == refStr {
				return
			}
		}
		this.M_escalationEventDefinitionPtr = append(this.M_escalationEventDefinitionPtr, ref.(string))
	}else{
		this.RemoveEscalationEventDefinitionPtr(ref)
		this.m_escalationEventDefinitionPtr = append(this.m_escalationEventDefinitionPtr, ref.(*EscalationEventDefinition))
		this.M_escalationEventDefinitionPtr = append(this.M_escalationEventDefinitionPtr, ref.(BaseElement).GetUUID())
	}
}

/** Remove reference EscalationEventDefinition **/
func (this *Escalation) RemoveEscalationEventDefinitionPtr(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	escalationEventDefinitionPtr_ := make([]*EscalationEventDefinition, 0)
	escalationEventDefinitionPtrUuid := make([]string, 0)
	for i := 0; i < len(this.m_escalationEventDefinitionPtr); i++ {
		if toDelete.GetUUID() != this.m_escalationEventDefinitionPtr[i].GetUUID() {
			escalationEventDefinitionPtr_ = append(escalationEventDefinitionPtr_, this.m_escalationEventDefinitionPtr[i])
			escalationEventDefinitionPtrUuid = append(escalationEventDefinitionPtrUuid, this.M_escalationEventDefinitionPtr[i])
		}
	}
	this.m_escalationEventDefinitionPtr = escalationEventDefinitionPtr_
	this.M_escalationEventDefinitionPtr = escalationEventDefinitionPtrUuid
}

/** Lane **/
func (this *Escalation) GetLanePtr() []*Lane{
	return this.m_lanePtr
}

/** Init reference Lane **/
func (this *Escalation) SetLanePtr(ref interface{}){
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
func (this *Escalation) RemoveLanePtr(ref interface{}){
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
func (this *Escalation) GetOutgoingPtr() []*Association{
	return this.m_outgoingPtr
}

/** Init reference Outgoing **/
func (this *Escalation) SetOutgoingPtr(ref interface{}){
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
func (this *Escalation) RemoveOutgoingPtr(ref interface{}){
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
func (this *Escalation) GetIncomingPtr() []*Association{
	return this.m_incomingPtr
}

/** Init reference Incoming **/
func (this *Escalation) SetIncomingPtr(ref interface{}){
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
func (this *Escalation) RemoveIncomingPtr(ref interface{}){
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

/** Definitions **/
func (this *Escalation) GetDefinitionsPtr() *Definitions{
	return this.m_definitionsPtr
}

/** Init reference Definitions **/
func (this *Escalation) SetDefinitionsPtr(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_definitionsPtr = ref.(string)
	}else{
		this.m_definitionsPtr = ref.(*Definitions)
		this.M_definitionsPtr = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference Definitions **/
func (this *Escalation) RemoveDefinitionsPtr(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_definitionsPtr.GetUUID() {
		this.m_definitionsPtr = nil
		this.M_definitionsPtr = ""
	}
}
