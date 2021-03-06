package Utility

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unsafe"

	"github.com/pborman/uuid"
	"github.com/robertkrimen/otto"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	/*
		A JavaScript identifier must start with a letter, underscore (_), or dollar sign ($);
		subsequent characters can also be digits (0-9).
		Because JavaScript is case sensitive, letters include the characters "A"
		through "Z" (uppercase) and the characters "a" through "z" (lowercase).
	*/
	UUID_PATTERN               = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
	VARIABLE_NAME_PATTERN      = "^[a-zA-Z_$][a-zA-Z_$0-9]*$"
	PACKAGE_NAME_PATTERN       = "^[a-zA-Z_$][a-zA-Z_$0-9]*(\\.[a-zA-Z_$][a-zA-Z_$0-9]*)+(\\.[a-zA-Z_$][a-zA-Z_$0-9]*)*$"
	ENTITY_NAME_PATTERN        = "^[a-zA-Z_$][a-zA-Z_$0-9]*(\\.[a-zA-Z_$][a-zA-Z_$0-9]*)+(\\.[a-zA-Z_$][a-zA-Z_$0-9]*)*\\%[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
	ISO_8601_TIME_PATTERN      = `^(?P<hour>2[0-3]|[01][0-9]):(?P<minute>[0-5][0-9]):(?P<second>[0-5][0-9])(?P<ms>\.[0-9]+)?(?P<timezone>Z|[+-](?:2[0-3]|[01][0-9]):[0-5][0-9])?$`
	ISO_8601_DATE_PATTERN      = `^(?P<year>-?(?:[1-9][0-9]*)?[0-9]{4})-(?P<month>1[0-2]|0[1-9])-(?P<day>3[01]|0[1-9]|[12][0-9])$`
	ISO_8601_DATE_TIME_PATTERN = `^(?P<year>-?(?:[1-9][0-9]*)?[0-9]{4})-(?P<month>1[0-2]|0[1-9])-(?P<day>3[01]|0[1-9]|[12][0-9])T(?P<hour>2[0-3]|[01][0-9]):(?P<minute>[0-5][0-9]):(?P<second>[0-5][0-9])(?P<ms>\.[0-9]+)?(?P<timezone>Z|[+-](?:2[0-3]|[01][0-9]):[0-5][0-9])?$`
	URI_BASE_64_PATTERN        = `(data:)(\\w+)(\\/)(\\w+)(;base64)`
	STD_BASE_64_PATTERN        = `^(?:[A-Za-z0-9+/]{4})+(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$`
)

/** Utility function **/
func Contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func Remove(s []string, index int) ([]string, error) {
	if index >= len(s) {
		return nil, errors.New("Out of Range Error")
	}
	return append(s[:index], s[index+1:]...), nil
}

//Pretty print the result.
func PrettyPrint(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	return out.Bytes(), err
}

func ToJson(obj interface{}) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	var b_ []byte
	b_, err = PrettyPrint(b)
	if err != nil {
		return "", err
	}

	return string(b_), nil
}

////////////////////////////////////////////////////////////////////////////////
//              			Utility function...
////////////////////////////////////////////////////////////////////////////////
func MakeTimestamp() int64 {
	return time.Now().Unix()
}

func BytesToString(b []byte) string {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := reflect.StringHeader{bh.Data, bh.Len}
	return *(*string)(unsafe.Pointer(&sh))
}

func StringToBytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{sh.Data, sh.Len, 0}
	return *(*[]byte)(unsafe.Pointer(&bh))
}

func DateTimeFromString(str string, layout string) (time.Time, error) {
	return time.Parse(layout, str)
}

/**
 * Parse and return a time object from a 8601 iso string, the time zone is
 * the UTC.
 */
func MatchISO8601_Time(str string) (*time.Time, error) {
	var exp = regexp.MustCompile(ISO_8601_TIME_PATTERN)
	match := exp.FindStringSubmatch(str)
	if len(match) == 0 {
		return nil, errors.New(str + " now match iso 8601")
	}
	var hour, minute, second, miliSecond int
	for i, name := range exp.SubexpNames() {
		if i != 0 {
			if name == "hour" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				hour = int(val)
			} else if name == "minute" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				minute = int(val)
			} else if name == "second" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				second = int(val)
			} else if name == "ms" {
				val, _ := strconv.ParseFloat(match[i], 64)
				miliSecond = int(val * 1000)
			}
		}
	}
	// year/mounth/day all set to zero in that case.
	t := time.Date(0, time.Month(0), 0, hour, minute, second, miliSecond, time.UTC)
	return &t, nil
}

func MatchISO8601_Date(str string) (*time.Time, error) {
	var exp = regexp.MustCompile(ISO_8601_DATE_PATTERN)
	match := exp.FindStringSubmatch(str)
	if len(match) == 0 {
		return nil, errors.New(str + " not match iso 8601")
	}
	var year, month, day int
	for i, name := range exp.SubexpNames() {
		if i != 0 {
			if name == "year" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				year = int(val)
			} else if name == "month" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				month = int(val)
			} else if name == "day" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				day = int(val)
			}
		}
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return &t, nil
}

/**
 * Parse and return a time object from a 8601 iso string, the time zone is
 * the UTC.
 */
func MatchISO8601_DateTime(str string) (*time.Time, error) {
	var exp = regexp.MustCompile(ISO_8601_DATE_TIME_PATTERN)
	match := exp.FindStringSubmatch(str)
	if len(match) == 0 {
		return nil, errors.New(str + " not match iso 8601")
	}
	var year, month, day, hour, minute, second, miliSecond int
	for i, name := range exp.SubexpNames() {
		if i != 0 {
			if name == "year" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				year = int(val)
			} else if name == "month" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				month = int(val)
			} else if name == "day" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				day = int(val)
			} else if name == "hour" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				hour = int(val)
			} else if name == "minute" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				minute = int(val)
			} else if name == "second" {
				val, _ := strconv.ParseInt(match[i], 10, 64)
				second = int(val)
			} else if name == "ms" {
				val, _ := strconv.ParseFloat(match[i], 64)
				miliSecond = int(val * 1000)
			}
		}
	}
	t := time.Date(year, time.Month(month), day, hour, minute, second, miliSecond, time.UTC)
	return &t, nil
}

// Create a random uuid value.
func RandomUUID() string {
	return uuid.NewRandom().String()
}

// Create a MD5 hash value with UUID format.
func GenerateUUID(val string) string {
	return uuid.NewMD5(uuid.NameSpace_DNS, []byte(val)).String()
}

// Determine if a string is a UUID or not,
// a uuid is compose of a TypeName%UUID
func IsUuid(uuidStr string) bool {
	match, _ := regexp.MatchString(UUID_PATTERN, uuidStr)
	return match
}

// Determine if a string is a valid variable name
func IsValidVariableName(variableName string) bool {
	match, _ := regexp.MatchString(VARIABLE_NAME_PATTERN, variableName)
	return match
}

// Determine if a string is a valid package name
func IsValidPackageName(packageName string) bool {
	match, _ := regexp.MatchString(PACKAGE_NAME_PATTERN, packageName)
	return match
}

// Determine if a string is a valid entity reference name
func IsValidEntityReferenceName(entityReferenceName string) bool {
	match, _ := regexp.MatchString(ENTITY_NAME_PATTERN, entityReferenceName)
	return match
}

// Determine if a string is a valid base64 string
func IsStdBase64(str string) bool {
	if strings.HasPrefix(str, "/") {
		return false
	}
	match, _ := regexp.MatchString(STD_BASE_64_PATTERN, str)
	return match
}

func IsUriBase64(str string) bool {
	match, _ := regexp.MatchString(URI_BASE_64_PATTERN, str)
	return match
}

func CreateSha1Key(data []byte) string {
	h := sha1.New()
	h.Write([]byte(data))
	key := hex.EncodeToString(h.Sum(nil))
	return key
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func RemoveAccent(text string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ := transform.String(t, text)
	return s
}

/**
 * Recursive function that return the checksum value.
 */
func GetChecksum(values interface{}) string {
	var checksum string

	if reflect.TypeOf(values).String() == "map[string]interface {}" {
		var keys []string
		for k, _ := range values.(map[string]interface{}) {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if values.(map[string]interface{})[key] != nil {
				checksum += GetChecksum(values.(map[string]interface{})[key])
			}
		}

	} else if reflect.TypeOf(values).String() == "[]interface {}" {

		for i := 0; i < len(values.([]interface{})); i++ {
			if values.([]interface{})[i] != nil {
				checksum += GetChecksum(values.([]interface{})[i])
			}
		}

	} else if reflect.TypeOf(values).String() == "[]map[string]interface {}" {
		for i := 0; i < len(values.([]map[string]interface{})); i++ {
			if values.([]map[string]interface{})[i] != nil {
				checksum += GetChecksum(values.([]map[string]interface{})[i])
			}
		}
	} else if reflect.TypeOf(values).String() == "[]string" {
		for i := 0; i < len(values.([]string)); i++ {
			checksum += GetChecksum(values.([]string)[i])
		}
	} else if reflect.TypeOf(values).String() == "otto.Value" { // Call from JS...
		val := values.(otto.Value)
		// In that case I will use otto functionality to convert the value...
		if val.IsBoolean() {
			boolVal, _ := val.ToBoolean()
			checksum += ToString(boolVal)
		} else if val.IsNumber() {
			floatVal, _ := val.ToFloat()
			checksum += ToString(floatVal)
		} else if val.IsString() {
			strVal, _ := val.ToString()
			checksum += ToString(strVal)
		} else if val.IsObject() {
			obj, err := val.Export()
			if err == nil {
				// Use a recursion here...
				checksum += GetChecksum(obj)
			}
		}
	} else {
		// here the value must be a single value...
		checksum += ToString(values)
	}

	//log.Println(checksum)
	return GetMD5Hash(checksum)
}

// ToMap converts a struct to a map using the struct's tags.
//
// ToMap uses tags on struct fields to decide which fields to add to the
// returned map.
func ToMap(in interface{}) (map[string]interface{}, error) {
	jsonStr, err := json.Marshal(in)
	var out map[string]interface{}
	json.Unmarshal(jsonStr, &out)
	return out, err
}

const filechunk = 8192 // we settle for 8KB
func CreateFileChecksum(file *os.File) string {
	file.Seek(0, 0) // Set the reader back to the begenin of the file...
	// calculate the file size
	info, _ := file.Stat()
	filesize := info.Size()
	blocks := uint64(math.Ceil(float64(filesize) / float64(filechunk)))
	hash := md5.New()

	for i := uint64(0); i < blocks; i++ {
		blocksize := int(math.Min(filechunk, float64(filesize-int64(i*filechunk))))
		buf := make([]byte, blocksize)
		file.Read(buf)
		io.WriteString(hash, string(buf)) // append into the hash
	}
	checksum := hex.EncodeToString(hash.Sum(nil))
	file.Seek(0, 0) // Set the reader back to the begenin of the file...
	return checksum
}

// Exists reports whether the named file or directory exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func FileLine() string {
	_, fileName, fileLine, ok := runtime.Caller(1)
	var s string
	if ok {
		s = fmt.Sprintf("%s:%d", fileName, fileLine)
	} else {
		s = ""
	}
	return s
}

func FunctionName() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return f.Name()
}

/**
 * Insert a new string at a given position.
 */
func InsertStringAt(pos int, str string, arr *[]string) {
	*arr = append(*arr, "")
	for i := len(*arr) - 1; i > pos; i-- {
		(*arr)[i] = (*arr)[i-1]
	}
	(*arr)[pos] = str
}

func InsertIntAt(pos int, val int, arr *[]int) {
	*arr = append(*arr, 0)
	for i := len(*arr) - 1; i > pos; i-- {
		(*arr)[i] = (*arr)[i-1]
	}
	(*arr)[pos] = val
}

func InsertInt64At(pos int, val int64, arr *[]int64) {
	*arr = append(*arr, 0)
	for i := len(*arr) - 1; i > pos; i-- {
		(*arr)[i] = (*arr)[i-1]
	}
	(*arr)[pos] = val
}

func InsertBoolAt(pos int, val bool, arr *[]bool) {
	*arr = append(*arr, false)
	for i := len(*arr) - 1; i > pos; i-- {
		(*arr)[i] = (*arr)[i-1]
	}
	(*arr)[pos] = val
}

// IPInfo describes a particular IP address.
type IPInfo struct {
	// IP holds the described IP address.
	IP string
	// Hostname holds a DNS name associated with the IP address.
	Hostname string
	// City holds the city of the ISP location.
	City string
	// Country holds the two-letter country code.
	Country string
	// Loc holds the latitude and longitude of the
	// ISP location as a comma-separated northing, easting
	// pair of floating point numbers.
	Loc string
	// Org describes the organization that is
	// responsible for the IP address.
	Org string
	// Postal holds the post code or zip code region of the ISP location.
	Postal string
}

// MyIP provides information about the public IP address of the client.
func MyIP() (*IPInfo, error) {
	return ForeignIP("")
}

// ForeignIP provides information about the given IP address,
// which should be in dotted-quad form.
func ForeignIP(ip string) (*IPInfo, error) {
	if ip != "" {
		ip += "/" + ip
	}
	response, err := http.Get("http://ipinfo.io" + ip + "/json")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var ipinfo IPInfo
	if err := json.Unmarshal(contents, &ipinfo); err != nil {
		return nil, err
	}
	return &ipinfo, nil
}

// Various decoding function.

// Windows1250
func DecodeWindows1250(val string) (string, error) {

	b := []byte(val)
	dec := charmap.Windows1250.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// Windows1251
func DecodeWindows1251(val string) (string, error) {

	b := []byte(val)
	dec := charmap.Windows1251.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// Windows1252
func DecodeWindows1252(val string) (string, error) {

	b := []byte(val)
	dec := charmap.Windows1252.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// Windows1253
func DecodeWindows1253(val string) (string, error) {

	b := []byte(val)
	dec := charmap.Windows1253.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// Windows1254
func DecodeWindows1254(val string) (string, error) {

	b := []byte(val)
	dec := charmap.Windows1254.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// Windows1255
func DecodeWindows1255(val string) (string, error) {

	b := []byte(val)
	dec := charmap.Windows1255.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// Windows1256
func DecodeWindows1256(val string) (string, error) {

	b := []byte(val)
	dec := charmap.Windows1256.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// Windows1257
func DecodeWindows1257(val string) (string, error) {

	b := []byte(val)
	dec := charmap.Windows1257.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// Windows1258
func DecodeWindows1258(val string) (string, error) {

	b := []byte(val)
	dec := charmap.Windows1258.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_1
func DecodeISO8859_1(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_1.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_2
func DecodeISO8859_2(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_2.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_3
func DecodeISO8859_3(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_3.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_4
func DecodeISO8859_4(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_4.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_5
func DecodeISO8859_5(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_5.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_6
func DecodeISO8859_6(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_6.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_7
func DecodeISO8859_7(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_7.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_8
func DecodeISO8859_8(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_8.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_9
func DecodeISO8859_9(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_9.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_10
func DecodeISO8859_10(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_10.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_13
func DecodeISO8859_13(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_13.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_14
func DecodeISO8859_14(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_14.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_15
func DecodeISO8859_15(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_15.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// ISO8859_16
func DecodeISO8859_16(val string) (string, error) {

	b := []byte(val)
	dec := charmap.ISO8859_16.NewDecoder()
	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// KOI8R
func DecodeKOI8R(val string) (string, error) {

	b := []byte(val)
	dec := charmap.KOI8R.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

// KOI8U
func DecodeKOI8U(val string) (string, error) {

	b := []byte(val)
	dec := charmap.KOI8U.NewDecoder()

	// Take more space just in case some characters need
	// more bytes in UTF-8 than in Win1256.
	bUTF := make([]byte, len(b)*3)
	n, _, err := dec.Transform(bUTF, b, false)
	if err != nil {
		return "", err
	}

	bUTF = bUTF[:n]
	return string(bUTF), nil
}

/**
 * Convert a numerical value to a string.
 */
func ToString(value interface{}) string {
	var str string
	if reflect.TypeOf(value).Kind() == reflect.String {
		str += value.(string)
	} else if reflect.TypeOf(value).Kind() == reflect.Int {
		str += strconv.Itoa(toInt(value))
	} else if reflect.TypeOf(value).Kind() == reflect.Int8 {
		str += strconv.Itoa(int(value.(int8)))
	} else if reflect.TypeOf(value).Kind() == reflect.Int16 {
		str += strconv.Itoa(int(value.(int16)))
	} else if reflect.TypeOf(value).Kind() == reflect.Int32 {
		str += strconv.Itoa(int(value.(int32)))
	} else if reflect.TypeOf(value).Kind() == reflect.Int64 {
		str += strconv.Itoa(int(value.(int64)))
	} else if reflect.TypeOf(value).Kind() == reflect.Uint8 {
		str += strconv.Itoa(int(value.(uint8)))
	} else if reflect.TypeOf(value).Kind() == reflect.Uint16 {
		str += strconv.Itoa(int(value.(uint16)))
	} else if reflect.TypeOf(value).Kind() == reflect.Uint32 {
		str += strconv.Itoa(int(value.(uint32)))
	} else if reflect.TypeOf(value).Kind() == reflect.Uint64 {
		str += strconv.Itoa(int(value.(uint64)))
	} else if reflect.TypeOf(value).Kind() == reflect.Float32 {
		str += strconv.FormatFloat(float64(value.(float32)), 'f', -1, 32)
	} else if reflect.TypeOf(value).Kind() == reflect.Float64 {
		str += strconv.FormatFloat(value.(float64), 'f', -1, 64)
	} else if reflect.TypeOf(value).Kind() == reflect.Bool {
		str += strconv.FormatBool(value.(bool))
	} else if reflect.TypeOf(value).String() == "[]uint8" {
		str += string(value.([]uint8))
	} else {
		log.Panicln("Value with type:", reflect.TypeOf(value).String(), "cannot be convert to string")
	}
	// Remove leading space.
	return strings.TrimSpace(str)
}

func ToInt(value interface{}) int {
	var val int
	if reflect.TypeOf(value).Kind() == reflect.String {
		val, _ = strconv.Atoi(value.(string))
	} else if reflect.TypeOf(value).Kind() == reflect.Int {
		val = value.(int)
	} else if reflect.TypeOf(value).Kind() == reflect.Int8 {
		val = int(value.(int8))
	} else if reflect.TypeOf(value).Kind() == reflect.Int16 {
		val = int(value.(int16))
	} else if reflect.TypeOf(value).Kind() == reflect.Int32 {
		val = int(value.(int32))
	} else if reflect.TypeOf(value).Kind() == reflect.Int64 {
		val = int(value.(int64))
	} else if reflect.TypeOf(value).Kind() == reflect.Float32 {
		val = int(value.(float32))
	} else if reflect.TypeOf(value).Kind() == reflect.Float64 {
		val = int(value.(float64))
	} else if reflect.TypeOf(value).Kind() == reflect.Bool {
		if value.(bool) {
			val = 1
		} else {
			val = 0
		}
	} else if reflect.TypeOf(value).String() == "[]uint8" {
		val = int(binary.BigEndian.Uint64(value.([]uint8)))
	} else {
		log.Panicln("Value with type:", reflect.TypeOf(value).String(), "cannot be convert to integer value")
	}
	return val
}

func ToBool(value interface{}) bool {
	if reflect.TypeOf(value).Kind() == reflect.Bool {
		return value.(bool)
	} else if reflect.TypeOf(value).Kind() == reflect.String {
		value_, err := strconv.ParseBool(value.(string))
		if err != nil {
			return false
		} else {
			return value_
		}
	}
	return false
}
func ToNumeric(value interface{}) float64 {
	var val float64
	if reflect.TypeOf(value).Kind() == reflect.String {
		val, _ = strconv.ParseFloat(value.(string), 64)
	} else if reflect.TypeOf(value).Kind() == reflect.Int {
		val = float64(value.(int))
	} else if reflect.TypeOf(value).Kind() == reflect.Int8 {
		val = float64(int(value.(int8)))
	} else if reflect.TypeOf(value).Kind() == reflect.Int16 {
		val = float64(int(value.(int16)))
	} else if reflect.TypeOf(value).Kind() == reflect.Int32 {
		val = float64(int(value.(int32)))
	} else if reflect.TypeOf(value).Kind() == reflect.Int64 {
		val = float64(int(value.(int64)))
	} else if reflect.TypeOf(value).Kind() == reflect.Float32 {
		val = float64(value.(float32))
	} else if reflect.TypeOf(value).Kind() == reflect.Float64 {
		val = value.(float64)
	} else if reflect.TypeOf(value).Kind() == reflect.Bool {
		if value.(bool) {
			val = 1.0
		} else {
			val = 0.0
		}
	} else {
		log.Panicln("Value with type:", reflect.TypeOf(value).String(), "cannot be convert to numerical value")
	}
	return val
}

func Round(x float64, n int) float64 {
	pow := math.Pow(10, float64(n))
	if math.Abs(x*pow) > 1e17 {
		return x
	}
	v, frac := math.Modf(x * pow)
	if x > 0.0 {
		if frac > 0.5 || (frac == 0.5 && uint64(v)%2 != 0) {
			v += 1.0
		}
	} else {
		if frac < -0.5 || (frac == -0.5 && uint64(v)%2 != 0) {
			v -= 1.0
		}
	}
	return v / pow
}

func Less(val0 interface{}, val1 interface{}) bool {
	if val0 == nil || val1 == nil {
		return true
	}

	if reflect.TypeOf(val0).Kind() == reflect.String {
		return val0.(string) < val1.(string)
	} else if reflect.TypeOf(val0).Kind() == reflect.Int {
		return val0.(int) < val1.(int)
	} else if reflect.TypeOf(val0).Kind() == reflect.Int8 {
		return val0.(int8) < val1.(int8)
	} else if reflect.TypeOf(val0).Kind() == reflect.Int16 {
		return val0.(int16) < val1.(int16)
	} else if reflect.TypeOf(val0).Kind() == reflect.Int32 {
		return val0.(int32) < val1.(int32)
	} else if reflect.TypeOf(val0).Kind() == reflect.Int64 {
		return val0.(int64) < val1.(int64)
	} else if reflect.TypeOf(val0).Kind() == reflect.Float32 {
		return val0.(float32) < val1.(float32)
	} else if reflect.TypeOf(val0).Kind() == reflect.Float64 {
		return val0.(float64) < val1.(float64)
	} else {
		log.Println("Value with type:", reflect.TypeOf(val0).String(), "cannot be compare!")
	}
	return false
}

/**
 * Keep the parent node
 */
func GetFilePathsByExtension(path string, extension string) []string {
	files, err := ioutil.ReadDir(path)
	results := make([]string, 0)
	if err == nil {
		for i := 0; i < len(files); i++ {
			if files[i].IsDir() {
				results = append(results, GetFilePathsByExtension(path+"/"+files[i].Name(), extension)...)
			} else if strings.HasSuffix(files[i].Name(), extension) {
				results = append(results, path+"/"+files[i].Name())
			}
		}
	}
	return results
}

// Copy the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

// Write a string to a given file.
func WriteStringToFile(filepath, s string) error {
	fo, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fo.Close()

	_, err = io.Copy(fo, strings.NewReader(s))
	if err != nil {
		return err
	}

	return nil
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
