package GoJerryScript

//#cgo LDFLAGS:  -L/usr/local/lib -ljerry-core -ljerry-ext -ljerry-libm -ljerry-port-default
/*
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "jerryscript.h"
#include "jerryscript-ext/handler.h"
#include "jerryscript-debugger.h"

typedef jerry_value_t* jerry_value_p;
extern jerry_value_t call_function ( const jerry_value_t, const jerry_value_t, const jerry_value_p, jerry_size_t);
extern void setGoMethod(const char* name, jerry_value_t obj);
extern const char* get_object_reference_uuid(uintptr_t ref);
extern void delete_object_reference(uintptr_t ref);
extern jerry_value_t create_string (const char *str_p);
extern jerry_value_t eval (const char *source_p, size_t source_size, bool is_strict);
extern jerry_value_t create_error (jerry_error_t error_type, const char *message_p);
extern jerry_size_t string_to_char_buffer (const jerry_value_t value, char *buffer_p, size_t size);
extern jerry_size_t get_string_size (const jerry_value_t value);
extern void create_native_object(const char* uuid, jerry_value_t object);
extern jerry_value_t create_array (uint32_t);
extern jerry_value_t set_property_by_index (const jerry_value_t, uint32_t, const jerry_value_t);
extern jerry_value_t get_property_by_index (const jerry_value_t, uint32_t);
extern uint32_t get_array_length (const jerry_value_t);
extern jerry_value_t json_parse (const char *string_p, size_t string_size);
*/
import "C"

//import "reflect"
import "unsafe"
import "encoding/binary"
import "encoding/json"
import "math"
import "code.myceliUs.com/Utility"
import "errors"
import "reflect"
import "strings"
import "code.myceliUs.com/GoJavaScript"

//import "strconv"
import "log"

// Global variable.
var (
	// The global object.
	globalObj = Jerry_create_null()
)

// Return the global object pointer.
func getGlobalObject() Uint32_t {
	if Jerry_value_is_null(globalObj) {
		globalObj = Jerry_get_global_object()
	}
	return globalObj
}

// Set property.
func Jerry_set_object_property(obj Uint32_t, name string, value interface{}) error {
	propName := goToJs(name)
	var propValue Uint32_t
	if reflect.TypeOf(value).String() != "GoJerryScript.SwigcptrUint32_t" {
		propValue = goToJs(value)
		// non-object property...
		defer Jerry_release_value(propValue)
	} else {
		// In that case I will not release the property value
		// rigth now. The value will release when function call will go
		// out of context. In Case of global variable the release value must
		// be release explicitely.
		propValue = value.(Uint32_t)
	}

	// get the reuslt.
	setResult := Jerry_set_property(obj, propName, propValue)

	// Now I will release the isSet, propValue and propName.
	defer Jerry_release_value(propName)
	defer Jerry_release_value(setResult)

	if Jerry_value_is_error(setResult) {
		err := errors.New("fail to set property " + name)
		return err
	}

	return nil
}

func Jerry_get_object_property(obj Uint32_t, name string) Uint32_t {
	propName := goToJs(name)
	property := Jerry_get_property(obj, propName)

	return property
}

// Retrun true if an object own a given property.
func Jerry_object_own_property(obj Uint32_t, name string) bool {
	propName := goToJs(name)
	hasProperty := Jerry_has_own_property(obj, propName)

	// release ressource.
	defer Jerry_release_value(hasProperty)
	defer Jerry_release_value(propName)

	return Jerry_get_boolean_value(hasProperty)
}

// Eval a given script string.
func evalScript(script string) (GoJavaScript.Value, error) {

	// Now I will evaluate the function...
	cstr := C.CString(script)
	defer C.free(unsafe.Pointer(cstr))
	r := C.eval(cstr, C.size_t(len(script)), false)

	// Create a Uint_32 value from the result.
	// the ret object will be release in the NewValue function.
	ret := jerry_value_t_To_uint32_t(r)

	var value GoJavaScript.Value
	if Jerry_value_is_error(ret) {
		err := errors.New("Fail to run script " + script)
		defer Jerry_release_value(ret)
		log.Println(err)
		return value, err
	}

	// Here I will create the return value.
	value = *NewValue(ret)

	return value, nil
}

/**
 * Append a Js function to a given object.
 */
func appendJsFunction(object Uint32_t, name string, src string) error {
	// eval the script.
	_, err := evalScript(src)

	// in that case the function must be set as object function.
	if object != nil && err == nil {
		fct := Jerry_get_object_property(getGlobalObject(), name)
		if Jerry_value_is_function(fct) {
			// Set the function on the object.
			Jerry_set_object_property(object, name, fct)

			// remove it from the global object.
			if object != getGlobalObject() {
				Jerry_delete_property(getGlobalObject(), goToJs(name))
			}
		} else {
			return errors.New("no function found with name " + name)
		}
	}
	return err
}

/**
 * Set a Go function as a method on a given object.
 */
func setGoMethod(object Uint32_t, name string, fct interface{}) {
	if fct != nil {
		Utility.RegisterFunction(name, fct)
	}
	if Jerry_value_is_object(object) {
		C.setGoMethod(C.CString(name), uint32_t_To_Jerry_value_t(object))
	}
}

/**
 * Call a Js function / method
 */
func callJsFunction(obj Uint32_t, name string, params []interface{}) (GoJavaScript.Value, error) {

	var thisPtr C.jerry_value_t
	var fctPtr C.jerry_value_t
	var fct Uint32_t

	thisPtr = uint32_t_To_Jerry_value_t(obj)

	fct = Jerry_get_object_property(obj, name)
	defer Jerry_release_value(fct)

	fctPtr = uint32_t_To_Jerry_value_t(fct)
	var r Uint32_t

	var err error

	// if the function is define...
	if Jerry_value_is_function(fct) {
		// Now I will set the arguments...
		args := make([]C.jerry_value_t, len(params))
		for i := 0; i < len(params); i++ {
			if params[i] == nil {
				null := Jerry_create_null()
				defer Jerry_release_value(null)
				args[i] = uint32_t_To_Jerry_value_t(null)
			} else {
				p := goToJs(params[i])
				defer Jerry_release_value(p)
				args[i] = uint32_t_To_Jerry_value_t(p)
			}
		}

		var r_ C.jerry_value_t

		if len(args) > 0 {
			r_ = C.call_function(fctPtr, thisPtr, (C.jerry_value_p)(unsafe.Pointer(&args[0])), C.jerry_value_t(len(params)))
		} else {
			var args_ C.jerry_value_p
			r_ = C.call_function(fctPtr, thisPtr, args_, C.uint32_t(len(args)))
		}
		r = jerry_value_t_To_uint32_t(r_)
	} else {
		err = errors.New("Function " + name + " dosent exist")
	}

	if Jerry_value_is_error(r) {
		err = errors.New("Fail to call function " + name)
	}

	result := NewValue(r)
	return *result, err
}

//export object_native_free_callback
func object_native_free_callback(native_p C.uintptr_t) {

	uuid := C.GoString(C.get_object_reference_uuid(native_p))
	C.delete_object_reference(native_p)

	// Release the reference.
	Jerry_release_value(GoJavaScript.GetCache().GetJsObject(uuid).(Uint32_t))
	GoJavaScript.GetCache().RemoveObject(uuid)

	// Now I will ask the client side to remove it object reference to.
	GoJavaScript.CallGoFunction("Client", "DeleteGoObject", uuid)
}

// The handler is call directly from Jerry script and is use to connect JS and GO
//export handler
func handler(fct C.jerry_value_t, this C.jerry_value_t, args C.uintptr_t, length int) C.jerry_value_t {
	// The function pointer.
	fctPtr := jerry_value_t_To_uint32_t(fct)
	if Jerry_value_is_function(fctPtr) {
		proValue := Jerry_get_object_property(fctPtr, "name")
		defer Jerry_release_value(proValue)
		name, err := jsToGo(proValue)

		if err == nil {
			params := make([]interface{}, 0)
			for i := 0; i < length; i++ {
				// Create function parmeters.
				val, err := jsToGo((Uint32_t)(SwigcptrUint32_t(C.uintptr_t(args))))
				if err == nil {
					params = append(params, val)
				} else {
					log.Panicln(err)
					jsError := createError(JERRY_ERROR_COMMON, err.Error())
					return uint32_t_To_Jerry_value_t(jsError)
				}
				args += 4 // 32 bits integer.
			}

			// This is the owner of the function.
			thisPtr := jerry_value_t_To_uint32_t(this)
			if Jerry_value_is_object(thisPtr) {
				propUuid_ := Jerry_get_object_property(thisPtr, "uuid_")
				defer Jerry_release_value(propUuid_)
				uuid, err := jsToGo(propUuid_)
				if err == nil {
					result, err := GoJavaScript.CallGoFunction(uuid.(string), name.(string), params...)
					if err == nil && result != nil {
						jsVal := goToJs(result)
						return uint32_t_To_Jerry_value_t(jsVal)
					} else if err != nil {
						log.Panicln(err)
						jsError := createError(JERRY_ERROR_COMMON, err.Error())
						return uint32_t_To_Jerry_value_t(jsError)
					}
				} else {
					log.Panicln("---> uuid not found!")
				}

			} else {
				// There is no function owner I will simply call go function.
				result, err := GoJavaScript.CallGoFunction("", name.(string), params...)
				if err == nil && result != nil {
					jsVal := goToJs(result)
					return uint32_t_To_Jerry_value_t(jsVal)
				} else if err != nil {
					log.Panicln(err)
					jsError := createError(JERRY_ERROR_COMMON, err.Error())
					return uint32_t_To_Jerry_value_t(jsError)
				}
			}

		} else if err != nil {
			log.Panicln(err)
			jsError := createError(JERRY_ERROR_COMMON, err.Error())
			return uint32_t_To_Jerry_value_t(jsError)
		}
	}

	// here i will retrun a null value
	return uint32_t_To_Jerry_value_t(Jerry_create_undefined())
}

func uint32_t_To_Jerry_value_t(val Uint32_t) C.jerry_value_t {
	val_ := (*uintptr)(unsafe.Pointer(val.Swigcptr()))
	return C.jerry_value_t(*val_)
}

func jerry_value_t_To_uint32_t(val C.jerry_value_t) Uint32_t {
	return (Uint32_t)(SwigcptrUint32_t(C.uintptr_t((uintptr)(unsafe.Pointer(&val)))))

}

func float64ToByte(f float64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], math.Float64bits(f))
	return buf[:]
}

////////////// Uint 8 //////////////
// The Uint8 Type represent a 8 bit char.
type Uint8 struct {
	// The pointer that old the data.
	ptr unsafe.Pointer
}

/**
 * Free the values.
 */
func (self Uint8) Free() {
	C.free(unsafe.Pointer(self.ptr))
}

/**
 * Access the undelying memeory values pointer.
 */
func (self Uint8) Swigcptr() uintptr {
	return uintptr(self.ptr)
}

/**
 * Create an error message.
 */
func createError(errorType int, errorMsg string) Uint32_t {
	msg := C.CString(errorMsg)
	err := C.create_error(C.jerry_error_t(errorType), msg)
	defer C.free(unsafe.Pointer(msg))
	return jerry_value_t_To_uint32_t(err)
}

/**
 * Create a new JerryScript String from go string
 */
func newJsString(val string) Uint32_t {
	cstr := C.CString(val)
	defer C.free(unsafe.Pointer(cstr))
	str := C.create_string(cstr)
	return jerry_value_t_To_uint32_t(str)
}

/**
 * Create a new value and set it finalyse methode.
 */
func NewValue(ptr Uint32_t) *GoJavaScript.Value {
	// Here I will create a new GoJavaScript value.
	v := new(GoJavaScript.Value)
	v.TYPENAME = "GoJavaScript.Value"
	var err error
	// Export the value.
	v.Val, err = jsToGo(ptr)
	if err != nil {
		log.Println("---> error: ", err)
		return nil
	}
	return v
}

// Retreive an object by it uuid as a global object property.
func getJsObjectByUuid(uuid string) Uint32_t {

	// So here I will try to create a local Js representation of the object.
	objInfos, err := GoJavaScript.CallGoFunction("Client", "GetGoObjectInfos", uuid)

	if err == nil {
		// So here I got an object map info.
		// Create the object JS object.
		obj := Jerry_create_object()
		if !Jerry_value_is_object(obj) {
			log.Panicln("---> fail to create a new object! ", uuid)
		}

		// I will keep the reference in the js cache to be able to remove release
		// the c pointer reference latter.
		GoJavaScript.GetCache().SetJsObject(uuid, obj)

		// Set the uuid property.
		Jerry_set_object_property(obj, "uuid_", uuid)

		// Set native object to the object.
		C.create_native_object(C.CString(uuid), uint32_t_To_Jerry_value_t(obj))

		// Now I will set the object method.
		methods := objInfos.(map[string]interface{})["Methods"].(map[string]interface{})
		for name, src := range methods {
			if len(src.(string)) == 0 {
				// Set the go function here.
				cstr := C.CString(name)
				defer C.free(unsafe.Pointer(cstr))
				C.setGoMethod(C.CString(name), uint32_t_To_Jerry_value_t(obj))
			} else {
				// append the object function here.
				appendJsFunction(obj, name, src.(string))
			}
		}

		// I can remove the methods from the infos.
		delete(objInfos.(map[string]interface{}), "Methods")

		// Now the object properties.
		for name, value := range objInfos.(map[string]interface{}) {
			if reflect.TypeOf(value).Kind() == reflect.Slice {
				slice := reflect.ValueOf(value)
				values := jerry_value_t_To_uint32_t(C.create_array(C.uint32_t(slice.Len())))
				for i := 0; i < slice.Len(); i++ {
					e := slice.Index(i).Interface()
					if reflect.TypeOf(e).Kind() == reflect.Map {
						// Here The value contain a map... so I will append
						if e.(map[string]interface{})["TYPENAME"] != nil {
							if e.(map[string]interface{})["TYPENAME"].(string) == "GoJavaScript.ObjectRef" {
								value_ := getJsObjectByUuid(e.(map[string]interface{})["UUID"].(string))
								r := C.set_property_by_index(uint32_t_To_Jerry_value_t(values), C.uint32_t(uint32(i)), uint32_t_To_Jerry_value_t(goToJs(value_)))
								// Release the result
								Jerry_release_value(jerry_value_t_To_uint32_t(r))
							}
						} else {
							log.Println("---> unknow object propertie type 231")
						}
					} else {
						r := C.set_property_by_index(uint32_t_To_Jerry_value_t(values), C.uint32_t(uint32(i)), uint32_t_To_Jerry_value_t(goToJs(e)))
						// Release the result
						Jerry_release_value(jerry_value_t_To_uint32_t(r))
					}
				}
				Jerry_set_object_property(obj, name, values)

			} else if reflect.TypeOf(value).Kind() == reflect.Map {
				if value.(map[string]interface{})["TYPENAME"] != nil {
					if value.(map[string]interface{})["TYPENAME"].(string) == "GoJavaScript.ObjectRef" {
						value_ := getJsObjectByUuid(value.(map[string]interface{})["UUID"].(string))
						Jerry_set_object_property(obj, name, value_)
					} else {
						log.Println("---> unknow object propertie type 245")
					}
				}
			} else {
				// Standard object property, int, string, float...
				Jerry_set_object_property(obj, name, value)
			}
		}
		return obj
	}

	// The property is undefined.
	return nil
}

/**
 * Create a go string from a JS string pointer.
 */
func jsStrToGoStr(str Uint32_t) string {

	// Size info, ptr and it value
	str_ := uint32_t_To_Jerry_value_t(str)
	size := C.size_t(C.get_string_size(str_))

	buffer := (*C.char)(unsafe.Pointer(C.malloc(size)))

	// Test if the string is a valid utf8 string...
	C.string_to_char_buffer(uint32_t_To_Jerry_value_t(str), buffer, size)

	// Copy the value to a string.
	value := C.GoStringN(buffer, C.int(size))

	// free the buffer.
	C.free(unsafe.Pointer(buffer))

	return value
}

func goToJs(value interface{}) Uint32_t {
	var propValue Uint32_t
	var typeOf = reflect.TypeOf(value)

	if typeOf.Kind() == reflect.String {
		// String value
		propValue = newJsString(value.(string))
	} else if typeOf.Kind() == reflect.Bool {
		// Boolean value
		propValue = Jerry_create_boolean(value.(bool))
	} else if typeOf.Kind() == reflect.Int {
		propValue = Jerry_create_number(float64(value.(int)))
	} else if typeOf.Kind() == reflect.Int8 {
		propValue = Jerry_create_number(float64(value.(int8)))
	} else if typeOf.Kind() == reflect.Int16 {
		propValue = Jerry_create_number(float64(value.(int16)))
	} else if typeOf.Kind() == reflect.Int32 {
		propValue = Jerry_create_number(float64(value.(int32)))
	} else if typeOf.Kind() == reflect.Int64 {
		propValue = Jerry_create_number(float64(value.(int64)))
	} else if typeOf.Kind() == reflect.Uint {
		propValue = Jerry_create_number(float64(value.(uint)))
	} else if typeOf.Kind() == reflect.Uint8 {
		propValue = Jerry_create_number(float64(value.(uint8)))
	} else if typeOf.Kind() == reflect.Uint16 {
		propValue = Jerry_create_number(float64(value.(uint16)))
	} else if typeOf.Kind() == reflect.Uint32 {
		propValue = Jerry_create_number(float64(value.(uint32)))
	} else if reflect.TypeOf(value).Kind() == reflect.Uint64 {
		propValue = Jerry_create_number(float64(value.(uint64)))
	} else if typeOf.Kind() == reflect.Float32 {
		propValue = Jerry_create_number(float64(value.(float32)))
	} else if typeOf.Kind() == reflect.Float64 {
		propValue = Jerry_create_number(value.(float64))
	} else if typeOf.Kind() == reflect.Slice {
		// So here I will create a array and put value in it.
		s := reflect.ValueOf(value)
		l := uint32(s.Len())
		array := C.create_array(C.uint32_t(l))
		propValue = jerry_value_t_To_uint32_t(array)

		var i uint32
		for i = 0; i < l; i++ {
			v := goToJs(s.Index(int(i)).Interface())
			r := C.set_property_by_index(uint32_t_To_Jerry_value_t(propValue), C.uint32_t(i), uint32_t_To_Jerry_value_t(v))
			Jerry_release_value(jerry_value_t_To_uint32_t(r))
		}

	} else if typeOf.String() == "GoJerryScript.SwigcptrUint32_t" {
		// already a Uint32_t
		propValue = value.(Uint32_t)
	} else if typeOf.String() == "GoJavaScript.ObjectRef" {
		// I got a Js object reference.
		uuid := value.(GoJavaScript.ObjectRef).UUID
		propValue = getJsObjectByUuid(uuid)
	} else if typeOf.String() == "*GoJavaScript.ObjectRef" {
		// I got a Js object reference.
		uuid := value.(*GoJavaScript.ObjectRef).UUID
		propValue = getJsObjectByUuid(uuid)
	} else if typeOf.String() == "map[string]interface {}" {
		// In that case I will create a object from the value found in the map
		// and return it as prop value.
		data, err := json.Marshal(value)
		if err == nil {
			if value.(map[string]interface{})["TYPENAME"] != nil {
				ref, err := GoJavaScript.CallGoFunction("Client", "CreateGoObject", string(data))
				if err == nil {
					// In that case an object exist in the case...
					propValue = getJsObjectByUuid(ref.(*GoJavaScript.ObjectRef).UUID)
				}
			} else {
				// Not a registered type...
				cstr := C.CString(string(data))
				defer C.free(unsafe.Pointer(cstr))
				propValue = jerry_value_t_To_uint32_t(C.json_parse(cstr, C.size_t(len(string(data)))))
			}
		}
	} else if typeOf.Kind() == reflect.Struct {
		val, err := Utility.ToMap(value)
		if err == nil {
			return goToJs(val)
		}
	} else {
		log.Panicln("---> type not found ", value, typeOf.String())
	}

	return propValue
}

/**
 * Return equivalent value of a 32 bit c pointer.
 */
func jsToGo(input Uint32_t) (interface{}, error) {

	// the Go value...
	var value interface{}

	// Now I will get the result if any...
	if Jerry_value_is_null(input) {
		return nil, nil
	} else if Jerry_value_is_undefined(input) {
		return nil, nil
	} else if Jerry_value_is_error(input) {
		// In that case I will return the error.
		log.Println("----> error found!")
	} else if Jerry_value_is_number(input) {
		value = Jerry_get_number_value(input)
	} else if Jerry_value_is_string(input) {
		value = jsStrToGoStr(input)
	} else if Jerry_value_is_boolean(input) {
		value = Jerry_get_boolean_value(input)
	} else if Jerry_value_is_typedarray(input) {
		/** not implemented **/

	} else if Jerry_value_is_array(input) {
		count := (uint32)(C.get_array_length(uint32_t_To_Jerry_value_t(input)))
		// So here I got a array without type so I will get it property by index
		// and interpret each result.
		value = make([]interface{}, 0)
		var i uint32
		for i = 0; i < count; i++ {
			e := jerry_value_t_To_uint32_t(C.get_property_by_index(uint32_t_To_Jerry_value_t(input), C.uint32_t(i)))
			v, err := jsToGo(e)
			if err == nil {
				value = append(value.([]interface{}), v)
			}
		}
	} else if Jerry_value_is_object(input) {
		// The go object will be a copy of the Js object.
		if Jerry_object_own_property(input, "uuid_") {
			uuid_ := Jerry_get_object_property(input, "uuid_")
			defer Jerry_release_value(uuid_)

			// Get the uuid string.
			uuid, _ := jsToGo(uuid_)

			// Return and object reference.
			value = GoJavaScript.NewObjectRef(uuid.(string))
		} else {
			stringified := Jerry_json_stringfy(input)
			// if there is no error
			if !Jerry_value_is_error(stringified) {
				jsonStr := jsStrToGoStr(stringified)
				if strings.Index(jsonStr, "TYPENAME") != -1 {
					// So here I will create a remote action and tell the client to
					// create a Go object from jsonStr. The object will be set by
					// the client on the server.
					return GoJavaScript.CallGoFunction("Client", "CreateGoObject", jsonStr)
				}

				// In that case the object has no go representation...
				// and must be use only in JS.
				return nil, nil
			} else {
				// Continue any way with nil object instead of an error...
				return nil, nil //errors.New("fail to stringfy object!")
			}
		}
	} else if Jerry_value_is_function(input) {
		// Here a function is found
		log.Println("---> function found!", input)
	} else if Jerry_value_is_abort(input) {
		// Here a function is found
		log.Println("--->abort!", input)
	} else if Jerry_value_is_arraybuffer(input) {
		// Here a function is found
		log.Println("--->array buffer!", input)
	} else if Jerry_value_is_constructor(input) {
		// Here a function is found
		log.Println("--->constructor!", input)
	} else if Jerry_value_is_promise(input) {
		// Here a function is found
		log.Println("--->promise!", input)
	} else {
		log.Println("---> not implemented Jerry value type.")
	}

	return value, nil
}

////////////// Uint 16 //////////////
// The Uint16 Type represent a 16 bit char.
type Uint16 struct {
	// The pointer that old the data.
	ptr unsafe.Pointer
}

/**
 * Free the values.
 */
func (self Uint16) Free() {
	C.free(unsafe.Pointer(self.ptr))
}

/**
 * Access the undelying memeory values pointer.
 */
func (self Uint16) Swigcptr() uintptr {
	return uintptr(self.ptr)
}

////////////// Uint 32 //////////////
// The Uint32 Type represent a 32 bit char.
type Uint32 struct {
	// The pointer that old the data.
	ptr unsafe.Pointer
}

func NewUint32FromInt(i int32) Uint32 {
	var val Uint32
	val.ptr = unsafe.Pointer(&i)
	return val
}

/**
 * Free the values.
 */
func (self Uint32) Free() {
	C.free(unsafe.Pointer(self.ptr))
}

/**
 * Access the undelying memeory values pointer.
 */
func (self Uint32) Swigcptr() uintptr {
	return uintptr(self.ptr)
}

////////////// Instance //////////////

// Reference to an object.
type Instance struct {
	// The pointer that old the data.
	ptr unsafe.Pointer
}

func NewInstance(obj interface{}) Instance {
	var instance Instance
	return instance
}

/**
 * Free the values.
 */
func (self Instance) Free() {
	C.free(unsafe.Pointer(self.ptr))
}

/**
 * Access the undelying memeory values pointer.
 */
func (self Instance) Swigcptr() uintptr {
	return uintptr(self.ptr)
}

const sizeOfUintPtr = unsafe.Sizeof(uintptr(0))

func uintptrToBytes(u *uintptr) []byte {
	return (*[sizeOfUintPtr]byte)(unsafe.Pointer(u))[:]
}
