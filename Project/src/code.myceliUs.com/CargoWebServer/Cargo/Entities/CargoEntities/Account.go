// +build CargoEntities

package CargoEntities

import(
	"encoding/xml"
)

type Account struct{

	/** The entity UUID **/
	UUID string
	/** The entity TypeName **/
	TYPENAME string
	/** The parent uuid if there is some. **/
	ParentUuid string
	/** The relation name with the parent. **/
	ParentLnk string
	/** If the entity value has change... **/
	NeedSave bool
	/** Get entity by uuid function **/
	getEntityByUuid func(string)(interface{}, error)

	/** members of Entity **/
	M_id string

	/** members of Account **/
	M_name string
	M_password string
	M_email string
	M_sessions []*Session
	M_messages []Message
	m_userRef *User
	/** If the ref is a string and not an object **/
	M_userRef string
	m_rolesRef []*Role
	/** If the ref is a string and not an object **/
	M_rolesRef []string
	m_permissionsRef []*Permission
	/** If the ref is a string and not an object **/
	M_permissionsRef []string


	/** Associations **/
	m_entitiesPtr *Entities
	/** If the ref is a string and not an object **/
	M_entitiesPtr string
}

/** Xml parser for Account **/
type XsdAccount struct {
	XMLName xml.Name	`xml:"toRef"`
	/** Entity **/
	M_id	string	`xml:"id,attr"`


	M_userRef	*string	`xml:"userRef"`
	M_rolesRef	[]string	`xml:"rolesRef"`
	M_permissionsRef	[]string	`xml:"permissionsRef"`
	M_sessions	[]*XsdSession	`xml:"sessions,omitempty"`

	M_name	string	`xml:"name,attr"`
	M_password	string	`xml:"password,attr"`
	M_email	string	`xml:"email,attr"`

}
/***************** Entity **************************/

/** UUID **/
func (this *Account) GetUuid() string{
	return this.UUID
}
func (this *Account) SetUuid(uuid string){
	this.UUID = uuid
}

/** Return the array of entity id's without it uuid **/
func (this *Account) Ids() []interface{} {
	ids := make([]interface{}, 0)
	ids = append(ids, this.M_id)
	return ids
}

/** The type name **/
func (this *Account) GetTypeName() string{
	this.TYPENAME = "CargoEntities.Account"
	return this.TYPENAME
}

/** Return the entity parent UUID **/
func (this *Account) GetParentUuid() string{
	return this.ParentUuid
}

/** Set it parent UUID **/
func (this *Account) SetParentUuid(parentUuid string){
	this.ParentUuid = parentUuid
}

/** Return it relation with it parent, only one parent is possible by entity. **/
func (this *Account) GetParentLnk() string{
	return this.ParentLnk
}
func (this *Account) SetParentLnk(parentLnk string){
	this.ParentLnk = parentLnk
}

/** Evaluate if an entity needs to be saved. **/
func (this *Account) IsNeedSave() bool{
	return this.NeedSave
}

/** Give access to entity manager GetEntityByUuid function from Entities package. **/
func (this *Account) SetEntityGetter(fct func(uuid string)(interface{}, error)){
	this.getEntityByUuid = fct
}

/** Id **/
func (this *Account) GetId() string{
	return this.M_id
}

/** Init reference Id **/
func (this *Account) SetId(ref interface{}){
	if this.M_id != ref.(string) {
		this.M_id = ref.(string)
		this.NeedSave = true
	}
}

/** Remove reference Id **/

/** Name **/
func (this *Account) GetName() string{
	return this.M_name
}

/** Init reference Name **/
func (this *Account) SetName(ref interface{}){
	if this.M_name != ref.(string) {
		this.M_name = ref.(string)
		this.NeedSave = true
	}
}

/** Remove reference Name **/

/** Password **/
func (this *Account) GetPassword() string{
	return this.M_password
}

/** Init reference Password **/
func (this *Account) SetPassword(ref interface{}){
	if this.M_password != ref.(string) {
		this.M_password = ref.(string)
		this.NeedSave = true
	}
}

/** Remove reference Password **/

/** Email **/
func (this *Account) GetEmail() string{
	return this.M_email
}

/** Init reference Email **/
func (this *Account) SetEmail(ref interface{}){
	if this.M_email != ref.(string) {
		this.M_email = ref.(string)
		this.NeedSave = true
	}
}

/** Remove reference Email **/

/** Sessions **/
func (this *Account) GetSessions() []*Session{
	return this.M_sessions
}

/** Init reference Sessions **/
func (this *Account) SetSessions(ref interface{}){
	isExist := false
	var sessionss []*Session
	for i:=0; i<len(this.M_sessions); i++ {
		if this.M_sessions[i].GetUuid() != ref.(*Session).GetUuid() {
			sessionss = append(sessionss, this.M_sessions[i])
		} else {
			isExist = true
			sessionss = append(sessionss, ref.(*Session))
		}
	}
	if !isExist {
		sessionss = append(sessionss, ref.(*Session))
		this.NeedSave = true
		this.M_sessions = sessionss
	}
}

/** Remove reference Sessions **/
func (this *Account) RemoveSessions(ref interface{}){
	toDelete := ref.(*Session)
	sessions_ := make([]*Session, 0)
	for i := 0; i < len(this.M_sessions); i++ {
		if toDelete.GetUuid() != this.M_sessions[i].GetUuid() {
			sessions_ = append(sessions_, this.M_sessions[i])
		}else{
			this.NeedSave = true
		}
	}
	this.M_sessions = sessions_
}

/** Messages **/
func (this *Account) GetMessages() []Message{
	return this.M_messages
}

/** Init reference Messages **/
func (this *Account) SetMessages(ref interface{}){
	isExist := false
	var messagess []Message
	for i:=0; i<len(this.M_messages); i++ {
		if this.M_messages[i].(Entity).GetUuid() != ref.(Entity).GetUuid() {
			messagess = append(messagess, this.M_messages[i])
		} else {
			isExist = true
			messagess = append(messagess, ref.(Message))
		}
	}
	if !isExist {
		messagess = append(messagess, ref.(Message))
		this.NeedSave = true
		this.M_messages = messagess
	}
}

/** Remove reference Messages **/
func (this *Account) RemoveMessages(ref interface{}){
	toDelete := ref.(Entity)
	messages_ := make([]Message, 0)
	for i := 0; i < len(this.M_messages); i++ {
		if toDelete.GetUuid() != this.M_messages[i].(Entity).GetUuid() {
			messages_ = append(messages_, this.M_messages[i])
		}else{
			this.NeedSave = true
		}
	}
	this.M_messages = messages_
}

/** UserRef **/
func (this *Account) GetUserRef() *User{
	if this.m_userRef == nil {
		entity, err := this.getEntityByUuid(this.M_userRef)
		if err == nil {
			this.m_userRef = entity.(*User)
		}
	}
	return this.m_userRef
}
func (this *Account) GetUserRefStr() string{
	return this.M_userRef
}

/** Init reference UserRef **/
func (this *Account) SetUserRef(ref interface{}){
	if _, ok := ref.(string); ok {
		if this.M_userRef != ref.(string) {
			this.M_userRef = ref.(string)
			this.NeedSave = true
		}
	}else{
		if this.M_userRef != ref.(Entity).GetUuid() {
			this.M_userRef = ref.(Entity).GetUuid()
			this.NeedSave = true
		}
		this.m_userRef = ref.(*User)
	}
}

/** Remove reference UserRef **/
func (this *Account) RemoveUserRef(ref interface{}){
	toDelete := ref.(Entity)
	if this.m_userRef!= nil {
		if toDelete.GetUuid() == this.m_userRef.GetUuid() {
			this.m_userRef = nil
			this.M_userRef = ""
			this.NeedSave = true
		}
	}
}

/** RolesRef **/
func (this *Account) GetRolesRef() []*Role{
	if this.m_rolesRef == nil {
		this.m_rolesRef = make([]*Role, 0)
		for i := 0; i < len(this.M_rolesRef); i++ {
			entity, err := this.getEntityByUuid(this.M_rolesRef[i])
			if err == nil {
				this.m_rolesRef = append(this.m_rolesRef, entity.(*Role))
			}
		}
	}
	return this.m_rolesRef
}
func (this *Account) GetRolesRefStr() []string{
	return this.M_rolesRef
}

/** Init reference RolesRef **/
func (this *Account) SetRolesRef(ref interface{}){
	if refStr, ok := ref.(string); ok {
		for i:=0; i < len(this.M_rolesRef); i++ {
			if this.M_rolesRef[i] == refStr {
				return
			}
		}
		this.M_rolesRef = append(this.M_rolesRef, ref.(string))
		this.NeedSave = true
	}else{
		for i:=0; i < len(this.m_rolesRef); i++ {
			if this.m_rolesRef[i].GetUuid() == ref.(*Role).GetUuid() {
				return
			}
		}
		isExist := false
		for i:=0; i < len(this.M_rolesRef); i++ {
			if this.M_rolesRef[i] == ref.(*Role).GetUuid() {
				isExist = true
			}
		}
		this.m_rolesRef = append(this.m_rolesRef, ref.(*Role))
	if !isExist {
		this.M_rolesRef = append(this.M_rolesRef, ref.(*Role).GetUuid())
		this.NeedSave = true
	}
	}
}

/** Remove reference RolesRef **/
func (this *Account) RemoveRolesRef(ref interface{}){
	toDelete := ref.(*Role)
	rolesRef_ := make([]*Role, 0)
	rolesRefUuid := make([]string, 0)
	for i := 0; i < len(this.m_rolesRef); i++ {
		if toDelete.GetUuid() != this.m_rolesRef[i].GetUuid() {
			rolesRef_ = append(rolesRef_, this.m_rolesRef[i])
			rolesRefUuid = append(rolesRefUuid, this.M_rolesRef[i])
		}else{
			this.NeedSave = true
		}
	}
	this.m_rolesRef = rolesRef_
	this.M_rolesRef = rolesRefUuid
}

/** PermissionsRef **/
func (this *Account) GetPermissionsRef() []*Permission{
	if this.m_permissionsRef == nil {
		this.m_permissionsRef = make([]*Permission, 0)
		for i := 0; i < len(this.M_permissionsRef); i++ {
			entity, err := this.getEntityByUuid(this.M_permissionsRef[i])
			if err == nil {
				this.m_permissionsRef = append(this.m_permissionsRef, entity.(*Permission))
			}
		}
	}
	return this.m_permissionsRef
}
func (this *Account) GetPermissionsRefStr() []string{
	return this.M_permissionsRef
}

/** Init reference PermissionsRef **/
func (this *Account) SetPermissionsRef(ref interface{}){
	if refStr, ok := ref.(string); ok {
		for i:=0; i < len(this.M_permissionsRef); i++ {
			if this.M_permissionsRef[i] == refStr {
				return
			}
		}
		this.M_permissionsRef = append(this.M_permissionsRef, ref.(string))
		this.NeedSave = true
	}else{
		for i:=0; i < len(this.m_permissionsRef); i++ {
			if this.m_permissionsRef[i].GetUuid() == ref.(*Permission).GetUuid() {
				return
			}
		}
		isExist := false
		for i:=0; i < len(this.M_permissionsRef); i++ {
			if this.M_permissionsRef[i] == ref.(*Permission).GetUuid() {
				isExist = true
			}
		}
		this.m_permissionsRef = append(this.m_permissionsRef, ref.(*Permission))
	if !isExist {
		this.M_permissionsRef = append(this.M_permissionsRef, ref.(*Permission).GetUuid())
		this.NeedSave = true
	}
	}
}

/** Remove reference PermissionsRef **/
func (this *Account) RemovePermissionsRef(ref interface{}){
	toDelete := ref.(*Permission)
	permissionsRef_ := make([]*Permission, 0)
	permissionsRefUuid := make([]string, 0)
	for i := 0; i < len(this.m_permissionsRef); i++ {
		if toDelete.GetUuid() != this.m_permissionsRef[i].GetUuid() {
			permissionsRef_ = append(permissionsRef_, this.m_permissionsRef[i])
			permissionsRefUuid = append(permissionsRefUuid, this.M_permissionsRef[i])
		}else{
			this.NeedSave = true
		}
	}
	this.m_permissionsRef = permissionsRef_
	this.M_permissionsRef = permissionsRefUuid
}

/** Entities **/
func (this *Account) GetEntitiesPtr() *Entities{
	if this.m_entitiesPtr == nil {
		entity, err := this.getEntityByUuid(this.M_entitiesPtr)
		if err == nil {
			this.m_entitiesPtr = entity.(*Entities)
		}
	}
	return this.m_entitiesPtr
}
func (this *Account) GetEntitiesPtrStr() string{
	return this.M_entitiesPtr
}

/** Init reference Entities **/
func (this *Account) SetEntitiesPtr(ref interface{}){
	if _, ok := ref.(string); ok {
		if this.M_entitiesPtr != ref.(string) {
			this.M_entitiesPtr = ref.(string)
			this.NeedSave = true
		}
	}else{
		if this.M_entitiesPtr != ref.(*Entities).GetUuid() {
			this.M_entitiesPtr = ref.(*Entities).GetUuid()
			this.NeedSave = true
		}
		this.m_entitiesPtr = ref.(*Entities)
	}
}

/** Remove reference Entities **/
func (this *Account) RemoveEntitiesPtr(ref interface{}){
	toDelete := ref.(*Entities)
	if this.m_entitiesPtr!= nil {
		if toDelete.GetUuid() == this.m_entitiesPtr.GetUuid() {
			this.m_entitiesPtr = nil
			this.M_entitiesPtr = ""
			this.NeedSave = true
		}
	}
}
