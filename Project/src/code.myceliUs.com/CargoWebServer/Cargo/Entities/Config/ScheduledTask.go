// +build Config

package Config

import (
	"encoding/xml"
)

type ScheduledTask struct {

	/** The entity UUID **/
	UUID string
	/** The entity TypeName **/
	TYPENAME string
	/** The parent uuid if there is some. **/
	ParentUuid string
	/** If the entity value has change... **/
	NeedSave bool

	/** If the entity is fully initialyse **/
	IsInit bool

	/** members of Configuration **/
	M_id string

	/** members of ScheduledTask **/
	M_isActive       bool
	M_script         string
	M_startTime      int
	M_expirationTime int
	M_frequency      int
	M_frequencyType  FrequencyType
	M_offsets        []int
	M_iterations     int
	M_keepAlive      bool

	/** Associations **/
	m_parentPtr *Configurations
	/** If the ref is a string and not an object **/
	M_parentPtr string
}

/** Xml parser for ScheduledTask **/
type XsdScheduledTask struct {
	XMLName xml.Name `xml:"scheduledTask"`
	/** Configuration **/
	M_id string `xml:"id,attr"`

	M_isActive       bool   `xml:"isActive,attr"`
	M_script         string `xml:"script,attr"`
	M_startTime      int    `xml:"startTime,attr"`
	M_expirationTime int    `xml:"expirationTime,attr"`
	M_frequency      int    `xml:"frequency,attr"`
	M_frequencyType  string `xml:"frequencyType,attr"`
	M_offsets        []int  `xml:"offsets,attr"`
	M_iterations     int    `xml:"iterations,attr"`
	M_keepAlive      bool   `xml:"keepAlive,attr"`
}

/** UUID **/
func (this *ScheduledTask) GetUUID() string {
	return this.UUID
}

/** Id **/
func (this *ScheduledTask) GetId() string {
	return this.M_id
}

/** Init reference Id **/
func (this *ScheduledTask) SetId(ref interface{}) {
	this.NeedSave = true
	this.M_id = ref.(string)
}

/** Remove reference Id **/

/** IsActive **/
func (this *ScheduledTask) IsActive() bool {
	return this.M_isActive
}

/** Init reference IsActive **/
func (this *ScheduledTask) SetIsActive(ref interface{}) {
	this.NeedSave = true
	this.M_isActive = ref.(bool)
}

/** Remove reference IsActive **/

/** Script **/
func (this *ScheduledTask) GetScript() string {
	return this.M_script
}

/** Init reference Script **/
func (this *ScheduledTask) SetScript(ref interface{}) {
	this.NeedSave = true
	this.M_script = ref.(string)
}

/** Remove reference Script **/

/** StartTime **/
func (this *ScheduledTask) GetStartTime() int {
	return this.M_startTime
}

/** Init reference StartTime **/
func (this *ScheduledTask) SetStartTime(ref interface{}) {
	this.NeedSave = true
	this.M_startTime = ref.(int)
}

/** Remove reference StartTime **/

/** ExpirationTime **/
func (this *ScheduledTask) GetExpirationTime() int {
	return this.M_expirationTime
}

/** Init reference ExpirationTime **/
func (this *ScheduledTask) SetExpirationTime(ref interface{}) {
	this.NeedSave = true
	this.M_expirationTime = ref.(int)
}

/** Remove reference ExpirationTime **/

/** Frequency **/
func (this *ScheduledTask) GetFrequency() int {
	return this.M_frequency
}

/** Init reference Frequency **/
func (this *ScheduledTask) SetFrequency(ref interface{}) {
	this.NeedSave = true
	this.M_frequency = ref.(int)
}

/** Remove reference Frequency **/

/** FrequencyType **/
func (this *ScheduledTask) GetFrequencyType() FrequencyType {
	return this.M_frequencyType
}

/** Init reference FrequencyType **/
func (this *ScheduledTask) SetFrequencyType(ref interface{}) {
	this.NeedSave = true
	this.M_frequencyType = ref.(FrequencyType)
}

/** Remove reference FrequencyType **/

/** Offsets **/
func (this *ScheduledTask) GetOffsets() []int {
	return this.M_offsets
}

/** Init reference Offsets **/
func (this *ScheduledTask) SetOffsets(ref interface{}) {
	this.NeedSave = true
	isExist := false
	var offsetss []int
	for i := 0; i < len(this.M_offsets); i++ {
		if this.M_offsets[i] != ref.(int) {
			offsetss = append(offsetss, this.M_offsets[i])
		} else {
			isExist = true
			offsetss = append(offsetss, ref.(int))
		}
	}
	if !isExist {
		offsetss = append(offsetss, ref.(int))
	}
	this.M_offsets = offsetss
}

/** Remove reference Offsets **/

/** Iterations **/
func (this *ScheduledTask) GetIterations() int {
	return this.M_iterations
}

/** Init reference Iterations **/
func (this *ScheduledTask) SetIterations(ref interface{}) {
	this.NeedSave = true
	this.M_iterations = ref.(int)
}

/** Remove reference Iterations **/

/** KeepAlive **/
func (this *ScheduledTask) GetKeepAlive() bool {
	return this.M_keepAlive
}

/** Init reference KeepAlive **/
func (this *ScheduledTask) SetKeepAlive(ref interface{}) {
	this.NeedSave = true
	this.M_keepAlive = ref.(bool)
}

/** Remove reference KeepAlive **/

/** Parent **/
func (this *ScheduledTask) GetParentPtr() *Configurations {
	return this.m_parentPtr
}

/** Init reference Parent **/
func (this *ScheduledTask) SetParentPtr(ref interface{}) {
	this.NeedSave = true
	if _, ok := ref.(string); ok {
		this.M_parentPtr = ref.(string)
	} else {
		this.m_parentPtr = ref.(*Configurations)
		this.M_parentPtr = ref.(*Configurations).GetUUID()
	}
}

/** Remove reference Parent **/
func (this *ScheduledTask) RemoveParentPtr(ref interface{}) {
	toDelete := ref.(*Configurations)
	if this.m_parentPtr != nil {
		if toDelete.GetUUID() == this.m_parentPtr.GetUUID() {
			this.m_parentPtr = nil
			this.M_parentPtr = ""
		} else {
			this.NeedSave = true
		}
	}
}
