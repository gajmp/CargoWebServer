package GoJerryScript

import "reflect"
import "code.myceliUs.com/Utility"
import "errors"

import "log"

/**
 * Interface Js value as Go value.
 */
type Value struct {
	// The go value that reflect the js value.
	Val interface{}
}

/**
 * If the value is an object it can be use as so.
 */
func (self *Value) Object() *Object {
	if self.Val != nil {
		if reflect.TypeOf(self.Val).String() == "GoJerryScript.Object" {
			obj := self.Val.(Object)
			return &obj
		} else if reflect.TypeOf(self.Val).String() == "*GoJerryScript.Object" {
			return self.Val.(*Object)
		}
	}

	// not an object.
	return nil
}

/**
 * Export the Javascript value in Go.
 */
func (self *Value) Export() (interface{}, error) {
	// Depending of the side where the value is it will be export in Go or in JS.
	if self.Val != nil {
		if reflect.TypeOf(self.Val).String() == "GoJerryScript.ObjectRef" {
			ref := self.Val.(ObjectRef)
			if GetCache().GetObject(ref.UUID) != nil {
				self.Val = GetCache().GetObject(ref.UUID)
			} else {
				log.Println("---> fail to retreive object ", ref.UUID)
			}
		} else if reflect.TypeOf(self.Val).Kind() == reflect.Slice {
			// In case of a slice I will transform the object ref with it actual values.
			slice := reflect.ValueOf(self.Val)
			values := make([]interface{}, 0)
			for i := 0; i < slice.Len(); i++ {
				e := slice.Index(i)
				if e.IsValid() {
					if !e.IsNil() {
						if reflect.TypeOf(e.Interface()).String() == "GoJerryScript.ObjectRef" {
							ref := e.Interface().(ObjectRef)
							if GetCache().GetObject(ref.UUID) != nil {
								obj := GetCache().GetObject(ref.UUID)
								if obj != nil {
									values = append(values, obj)
								} else {
									log.Println("---> fail to retreive object ", ref.UUID)
								}
							}
						}
					}
				}
			}

			// Set the values...
			if len(values) > 0 {
				self.Val = values
			}
		}
	}
	// Return the value itself.
	return self.Val, nil
}

/**
 * Type validation function.
 */
func (self *Value) IsString() bool {
	if reflect.TypeOf(self.Val).Kind() == reflect.String {
		return true
	}
	return false
}

func (self *Value) ToString() (string, error) {
	if !self.IsString() {
		return "", errors.New("The value is not a string!")
	}
	return self.Val.(string), nil
}

func (self *Value) IsBoolean() bool {
	if reflect.TypeOf(self.Val).Kind() == reflect.Bool {
		return true
	}
	return false
}

func (self *Value) ToBoolean() (bool, error) {
	if !self.IsBoolean() {
		return false, errors.New("The value is not a boolean!")
	}
	return self.Val.(bool), nil
}

func (self *Value) IsNull() bool {
	return self.Val == nil
}

func (self *Value) IsUndefined() bool {
	return self.Val == nil
}

func (self *Value) IsNumber() bool {
	if reflect.TypeOf(self.Val).Kind() == reflect.Float32 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Float64 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Int ||
		reflect.TypeOf(self.Val).Kind() == reflect.Int8 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Int16 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Int32 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Int64 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Uint ||
		reflect.TypeOf(self.Val).Kind() == reflect.Uint8 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Uint16 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Uint32 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Uint64 {
		return true
	}
	return false
}

func (self *Value) ToFloat() (float64, error) {
	if !self.IsNumber() {
		return 0.0, errors.New("The value is not a numeric!")
	}

	return self.Val.(float64), nil
}

func (self *Value) ToInteger() (int64, error) {
	if !(reflect.TypeOf(self.Val).Kind() == reflect.Int ||
		reflect.TypeOf(self.Val).Kind() == reflect.Int8 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Int16 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Int32 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Int64 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Uint ||
		reflect.TypeOf(self.Val).Kind() == reflect.Uint8 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Uint16 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Uint32 ||
		reflect.TypeOf(self.Val).Kind() == reflect.Uint64) {
		return 0, errors.New("The value is not an Interger!")
	}

	return int64(Utility.ToInt(self.Val)), nil
}