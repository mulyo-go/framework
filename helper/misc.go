package helper

import (
	"fmt"
	"reflect"
	"time"
)

// Dump mencetak value ke stdout (debug)
func Dump(values ...interface{}) {
	for _, v := range values {
		fmt.Printf("[DUMP] %+v\n", v)
	}
}

// Dd mencetak value dan exit (debug die)
func Dd(values ...interface{}) {
	for _, v := range values {
		fmt.Printf("[DD] %+v\n", v)
	}
	panic("dd() called")
}

// Value mengembalikan value atau default
func Value(v interface{}, defaults ...interface{}) interface{} {
	if v != nil && v != "" {
		return v
	}
	if len(defaults) > 0 {
		return defaults[0]
	}
	return nil
}

// Optional membungkus value untuk safe access
type OptionalValue struct {
	value interface{}
}

func Optional(v interface{}) *OptionalValue {
	return &OptionalValue{value: v}
}

func (o *OptionalValue) Get(keys ...string) interface{} {
	if o == nil || o.value == nil {
		return nil
	}
	current := o.value
	for _, key := range keys {
		val := reflect.ValueOf(current)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() == reflect.Map {
			mapVal := val.MapIndex(reflect.ValueOf(key))
			if !mapVal.IsValid() {
				return nil
			}
			current = mapVal.Interface()
		} else if val.Kind() == reflect.Struct {
			field := val.FieldByName(key)
			if !field.IsValid() {
				return nil
			}
			current = field.Interface()
		} else {
			return nil
		}
	}
	return current
}

// Blank mengecek apakah value kosong
func Blank(v interface{}) bool {
	if v == nil {
		return true
	}
	s, ok := v.(string)
	if ok {
		return s == ""
	}
	return false
}

// Filled mengecek apakah value terisi
func Filled(v interface{}) bool {
	return !Blank(v)
}

// FormatIDR memformat angka ke format Rupiah
func FormatIDR(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return "Rp " + s
	}
	result := ""
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += "."
		}
		result += string(ch)
	}
	return "Rp " + result
}

// NowUnix returns current Unix timestamp in milliseconds (bigint 13 digit)
func NowUnix() int64 {
	return time.Now().UnixMilli()
}

// Ternary mengembalikan value berdasarkan kondisi
func Ternary(condition bool, trueVal, falseVal interface{}) interface{} {
	if condition {
		return trueVal
	}
	return falseVal
}

// Default mengembalikan default value jika v kosong
func Default(v interface{}, defaultVal interface{}) interface{} {
	if Blank(v) {
		return defaultVal
	}
	return v
}
