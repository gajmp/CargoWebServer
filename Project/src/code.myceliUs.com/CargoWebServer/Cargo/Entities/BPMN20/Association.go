// +build BPMN20

package BPMN20

import(
"encoding/xml"
)

type Association struct{

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

	/** members of Artifact **/
	/** No members **/

	/** members of Association **/
	M_associationDirection AssociationDirection
	m_sourceRef BaseElement
	/** If the ref is a string and not an object **/
	M_sourceRef string
	m_targetRef BaseElement
	/** If the ref is a string and not an object **/
	M_targetRef string


	/** Associations **/
	m_lanePtr []*Lane
	/** If the ref is a string and not an object **/
	M_lanePtr []string
	m_outgoingPtr []*Association
	/** If the ref is a string and not an object **/
	M_outgoingPtr []string
	m_incomingPtr []*Association
	/** If the ref is a string and not an object **/
	M_incomingPtr []string
	m_processPtr *Process
	/** If the ref is a string and not an object **/
	M_processPtr string
	m_collaborationPtr Collaboration
	/** If the ref is a string and not an object **/
	M_collaborationPtr string
	m_subChoreographyPtr *SubChoreography
	/** If the ref is a string and not an object **/
	M_subChoreographyPtr string
	m_subProcessPtr SubProcess
	/** If the ref is a string and not an object **/
	M_subProcessPtr string
}

/** Xml parser for Association **/
type XsdAssociation struct {
	XMLName xml.Name	`xml:"association"`
	/** BaseElement **/
	M_documentation	[]*XsdDocumentation	`xml:"documentation,omitempty"`
	M_extensionElements	*XsdExtensionElements	`xml:"extensionElements,omitempty"`
	M_id	string	`xml:"id,attr"`
//	M_other	string	`xml:",innerxml"`


	/** Artifact **/


	M_sourceRef	string	`xml:"sourceRef,attr"`
	M_targetRef	string	`xml:"targetRef,attr"`
	M_associationDirection	string	`xml:"associationDirection,attr"`

}
/** UUID **/
func (this *Association) GetUUID() string{
	return this.UUID
}

/** Id **/
func (this *Association) GetId() string{
	return this.M_id
}

/** Init reference Id **/
func (this *Association) SetId(ref interface{}){
	this.NeedSave = true
	this.M_id = ref.(string)
}

/** Remove reference Id **/

/** Other **/
func (this *Association) GetOther() interface{}{
	return this.M_other
}

/** Init reference Other **/
func (this *Association) SetOther(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_other = ref.(string)
	}else{
		this.m_other = ref.(interface{})
	}
}

/** Remove reference Other **/

/** ExtensionElements **/
func (this *Association) GetExtensionElements() *ExtensionElements{
	return this.M_extensionElements
}

/** Init reference ExtensionElements **/
func (this *Association) SetExtensionElements(ref interface{}){
	this.NeedSave = true
	this.M_extensionElements = ref.(*ExtensionElements)
}

/** Remove reference ExtensionElements **/
func (this *Association) RemoveExtensionElements(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(*ExtensionElements)
	if toDelete.GetUUID() == this.M_extensionElements.GetUUID() {
		this.M_extensionElements = nil
	}
}

/** ExtensionDefinitions **/
func (this *Association) GetExtensionDefinitions() []*ExtensionDefinition{
	return this.M_extensionDefinitions
}

/** Init reference ExtensionDefinitions **/
func (this *Association) SetExtensionDefinitions(ref interface{}){
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
func (this *Association) RemoveExtensionDefinitions(ref interface{}){
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
func (this *Association) GetExtensionValues() []*ExtensionAttributeValue{
	return this.M_extensionValues
}

/** Init reference ExtensionValues **/
func (this *Association) SetExtensionValues(ref interface{}){
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
func (this *Association) RemoveExtensionValues(ref interface{}){
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
func (this *Association) GetDocumentation() []*Documentation{
	return this.M_documentation
}

/** Init reference Documentation **/
func (this *Association) SetDocumentation(ref interface{}){
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
func (this *Association) RemoveDocumentation(ref interface{}){
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

/** AssociationDirection **/
func (this *Association) GetAssociationDirection() AssociationDirection{
	return this.M_associationDirection
}

/** Init reference AssociationDirection **/
func (this *Association) SetAssociationDirection(ref interface{}){
	this.NeedSave = true
	this.M_associationDirection = ref.(AssociationDirection)
}

/** Remove reference AssociationDirection **/

/** SourceRef **/
func (this *Association) GetSourceRef() BaseElement{
	return this.m_sourceRef
}

/** Init reference SourceRef **/
func (this *Association) SetSourceRef(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_sourceRef = ref.(string)
	}else{
		this.m_sourceRef = ref.(BaseElement)
		this.M_sourceRef = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference SourceRef **/
func (this *Association) RemoveSourceRef(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_sourceRef.(BaseElement).GetUUID() {
		this.m_sourceRef = nil
		this.M_sourceRef = ""
	}
}

/** TargetRef **/
func (this *Association) GetTargetRef() BaseElement{
	return this.m_targetRef
}

/** Init reference TargetRef **/
func (this *Association) SetTargetRef(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_targetRef = ref.(string)
	}else{
		this.m_targetRef = ref.(BaseElement)
		this.M_targetRef = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference TargetRef **/
func (this *Association) RemoveTargetRef(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_targetRef.(BaseElement).GetUUID() {
		this.m_targetRef = nil
		this.M_targetRef = ""
	}
}

/** Lane **/
func (this *Association) GetLanePtr() []*Lane{
	return this.m_lanePtr
}

/** Init reference Lane **/
func (this *Association) SetLanePtr(ref interface{}){
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
func (this *Association) RemoveLanePtr(ref interface{}){
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
func (this *Association) GetOutgoingPtr() []*Association{
	return this.m_outgoingPtr
}

/** Init reference Outgoing **/
func (this *Association) SetOutgoingPtr(ref interface{}){
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
func (this *Association) RemoveOutgoingPtr(ref interface{}){
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
func (this *Association) GetIncomingPtr() []*Association{
	return this.m_incomingPtr
}

/** Init reference Incoming **/
func (this *Association) SetIncomingPtr(ref interface{}){
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
func (this *Association) RemoveIncomingPtr(ref interface{}){
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

/** Process **/
func (this *Association) GetProcessPtr() *Process{
	return this.m_processPtr
}

/** Init reference Process **/
func (this *Association) SetProcessPtr(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_processPtr = ref.(string)
	}else{
		this.m_processPtr = ref.(*Process)
		this.M_processPtr = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference Process **/
func (this *Association) RemoveProcessPtr(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_processPtr.GetUUID() {
		this.m_processPtr = nil
		this.M_processPtr = ""
	}
}

/** Collaboration **/
func (this *Association) GetCollaborationPtr() Collaboration{
	return this.m_collaborationPtr
}

/** Init reference Collaboration **/
func (this *Association) SetCollaborationPtr(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_collaborationPtr = ref.(string)
	}else{
		this.m_collaborationPtr = ref.(Collaboration)
		this.M_collaborationPtr = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference Collaboration **/
func (this *Association) RemoveCollaborationPtr(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_collaborationPtr.(BaseElement).GetUUID() {
		this.m_collaborationPtr = nil
		this.M_collaborationPtr = ""
	}
}

/** SubChoreography **/
func (this *Association) GetSubChoreographyPtr() *SubChoreography{
	return this.m_subChoreographyPtr
}

/** Init reference SubChoreography **/
func (this *Association) SetSubChoreographyPtr(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_subChoreographyPtr = ref.(string)
	}else{
		this.m_subChoreographyPtr = ref.(*SubChoreography)
		this.M_subChoreographyPtr = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference SubChoreography **/
func (this *Association) RemoveSubChoreographyPtr(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_subChoreographyPtr.GetUUID() {
		this.m_subChoreographyPtr = nil
		this.M_subChoreographyPtr = ""
	}
}

/** SubProcess **/
func (this *Association) GetSubProcessPtr() SubProcess{
	return this.m_subProcessPtr
}

/** Init reference SubProcess **/
func (this *Association) SetSubProcessPtr(ref interface{}){
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_subProcessPtr = ref.(string)
	}else{
		this.m_subProcessPtr = ref.(SubProcess)
		this.M_subProcessPtr = ref.(BaseElement).GetUUID()
	}
}

/** Remove reference SubProcess **/
func (this *Association) RemoveSubProcessPtr(ref interface{}){
	this.NeedSave = true
	toDelete := ref.(BaseElement)
	if toDelete.GetUUID() == this.m_subProcessPtr.(BaseElement).GetUUID() {
		this.m_subProcessPtr = nil
		this.M_subProcessPtr = ""
	}
}
