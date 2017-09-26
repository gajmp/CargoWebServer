// +build Config

package Config

import(
	"encoding/xml"
)

type SmtpConfiguration struct{

	/** The entity UUID **/
	UUID string
	/** The entity TypeName **/
	TYPENAME string
	/** The parent uuid if there is some. **/
	ParentUuid string
	/** If the entity value has change... **/
	NeedSave bool

	/** If the entity is fully initialyse **/
	IsInit   bool

	/** members of Configuration **/
	M_id string

	/** members of SmtpConfiguration **/
	M_textEncoding Encoding
	M_hostName string
	M_ipv4 string
	M_port int
	M_user string
	M_pwd string


	/** Associations **/
	m_parentPtr *Configurations
	/** If the ref is a string and not an object **/
	M_parentPtr string
}

/** Xml parser for SmtpConfiguration **/
type XsdSmtpConfiguration struct {
	XMLName xml.Name	`xml:"smtpConfiguration"`
	/** Configuration **/
	M_id	string	`xml:"id,attr"`


	M_hostName	string	`xml:"hostName,attr"`
	M_ipv4	string	`xml:"ipv4,attr"`
	M_port	int	`xml:"port,attr"`
	M_user	string	`xml:"user,attr"`
	M_pwd	string	`xml:"pwd,attr"`
	M_textEncoding	string	`xml:"textEncoding,attr"`

}
/** UUID **/
func (this *SmtpConfiguration) GetUUID() string{
	return this.UUID
}

/** Id **/
func (this *SmtpConfiguration) GetId() string{
	return this.M_id
}

/** Init reference Id **/
func (this *SmtpConfiguration) SetId(ref interface{}){
	this.M_id = ref.(string)
}

/** Remove reference Id **/

/** TextEncoding **/
func (this *SmtpConfiguration) GetTextEncoding() Encoding{
	return this.M_textEncoding
}

/** Init reference TextEncoding **/
func (this *SmtpConfiguration) SetTextEncoding(ref interface{}){
	this.M_textEncoding = ref.(Encoding)
}

/** Remove reference TextEncoding **/

/** HostName **/
func (this *SmtpConfiguration) GetHostName() string{
	return this.M_hostName
}

/** Init reference HostName **/
func (this *SmtpConfiguration) SetHostName(ref interface{}){
	this.M_hostName = ref.(string)
}

/** Remove reference HostName **/

/** Ipv4 **/
func (this *SmtpConfiguration) GetIpv4() string{
	return this.M_ipv4
}

/** Init reference Ipv4 **/
func (this *SmtpConfiguration) SetIpv4(ref interface{}){
	this.M_ipv4 = ref.(string)
}

/** Remove reference Ipv4 **/

/** Port **/
func (this *SmtpConfiguration) GetPort() int{
	return this.M_port
}

/** Init reference Port **/
func (this *SmtpConfiguration) SetPort(ref interface{}){
	this.M_port = ref.(int)
}

/** Remove reference Port **/

/** User **/
func (this *SmtpConfiguration) GetUser() string{
	return this.M_user
}

/** Init reference User **/
func (this *SmtpConfiguration) SetUser(ref interface{}){
	this.M_user = ref.(string)
}

/** Remove reference User **/

/** Pwd **/
func (this *SmtpConfiguration) GetPwd() string{
	return this.M_pwd
}

/** Init reference Pwd **/
func (this *SmtpConfiguration) SetPwd(ref interface{}){
	this.M_pwd = ref.(string)
}

/** Remove reference Pwd **/

/** Parent **/
func (this *SmtpConfiguration) GetParentPtr() *Configurations{
	return this.m_parentPtr
}

/** Init reference Parent **/
func (this *SmtpConfiguration) SetParentPtr(ref interface{}){
	if _, ok := ref.(string); ok {
		this.M_parentPtr = ref.(string)
	}else{
		this.M_parentPtr = ref.(*Configurations).GetUUID()
		this.m_parentPtr = ref.(*Configurations)
	}
}

/** Remove reference Parent **/
func (this *SmtpConfiguration) RemoveParentPtr(ref interface{}){
	toDelete := ref.(*Configurations)
	if this.m_parentPtr!= nil {
		if toDelete.GetUUID() == this.m_parentPtr.GetUUID() {
			this.m_parentPtr = nil
			this.M_parentPtr = ""
		}
	}
}
