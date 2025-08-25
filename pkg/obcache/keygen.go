package obcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// DefaultKeyFunc generates cache keys from function arguments using a hash-based approach
// This function handles most common Go types and provides stable key generation
func DefaultKeyFunc(args []any) string {
	if len(args) == 0 {
		return "no-args"
	}

	var parts []string
	for i, arg := range args {
		key := argToKey(arg)
		parts = append(parts, fmt.Sprintf("%d:%s", i, key))
	}

	// For short keys, return directly
	combined := strings.Join(parts, "|")
	if len(combined) <= 64 {
		return combined
	}

	// For longer keys, use SHA256 hash to prevent unbounded key growth
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// SimpleKeyFunc generates simple cache keys by joining string representations
// This is faster but may have collisions for complex types
func SimpleKeyFunc(args []any) string {
	if len(args) == 0 {
		return "no-args"
	}

	var parts []string
	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%v", arg))
	}

	return strings.Join(parts, ":")
}

// argToKey converts a single argument to a string key
func argToKey(arg any) string {
	if arg == nil {
		return "nil"
	}

	v := reflect.ValueOf(arg)
	t := v.Type()

	// Handle common types efficiently
	switch t.Kind() {
	case reflect.String:
		return "s:" + v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "i:" + strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "u:" + strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return "f:" + strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case reflect.Bool:
		return "b:" + strconv.FormatBool(v.Bool())
	case reflect.Ptr:
		if v.IsNil() {
			return "ptr:nil"
		}
		return "ptr:" + argToKey(v.Elem().Interface())
	case reflect.Slice, reflect.Array:
		return handleSliceOrArray(v)
	case reflect.Map:
		return handleMap(v)
	case reflect.Struct:
		return handleStruct(v, t)
	case reflect.Interface:
		if v.IsNil() {
			return "iface:nil"
		}
		return "iface:" + argToKey(v.Elem().Interface())
	default:
		// Fallback to string representation for other types
		return fmt.Sprintf("%T:%v", arg, arg)
	}
}

// handleSliceOrArray generates keys for slices and arrays
func handleSliceOrArray(v reflect.Value) string {
	if v.IsNil() {
		return "slice:nil"
	}

	length := v.Len()
	if length == 0 {
		return "slice:empty"
	}

	// For small slices, include all elements
	if length <= 10 {
		var elements []string
		for i := 0; i < length; i++ {
			elements = append(elements, argToKey(v.Index(i).Interface()))
		}
		return "slice:[" + strings.Join(elements, ",") + "]"
	}

	// For large slices, use length + hash of first and last elements
	first := argToKey(v.Index(0).Interface())
	last := argToKey(v.Index(length - 1).Interface())
	return fmt.Sprintf("slice:len%d:%s...%s", length, first, last)
}

// handleMap generates keys for maps
func handleMap(v reflect.Value) string {
	if v.IsNil() {
		return "map:nil"
	}

	keys := v.MapKeys()
	if len(keys) == 0 {
		return "map:empty"
	}

	// For small maps, include key-value pairs (sorted by key string representation for consistency)
	if len(keys) <= 5 {
		var pairs []string
		keyStrs := make([]string, len(keys))
		for i, key := range keys {
			keyStrs[i] = argToKey(key.Interface())
		}

		// Simple sort for consistency
		for i := 0; i < len(keyStrs); i++ {
			for j := i + 1; j < len(keyStrs); j++ {
				if keyStrs[i] > keyStrs[j] {
					keyStrs[i], keyStrs[j] = keyStrs[j], keyStrs[i]
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}

		for i, key := range keys {
			keyStr := keyStrs[i]
			valueStr := argToKey(v.MapIndex(key).Interface())
			pairs = append(pairs, keyStr+"="+valueStr)
		}
		return "map:{" + strings.Join(pairs, ",") + "}"
	}

	// For large maps, use length + type info
	return fmt.Sprintf("map:len%d:type%s", len(keys), v.Type().String())
}

// handleStruct generates keys for structs
func handleStruct(v reflect.Value, t reflect.Type) string {
	numFields := v.NumField()
	if numFields == 0 {
		return "struct:empty"
	}

	var fields []string
	for i := 0; i < numFields && i < 10; i++ { // Limit to first 10 fields
		field := t.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldValue := v.Field(i)
		if !fieldValue.CanInterface() {
			continue
		}

		fieldKey := argToKey(fieldValue.Interface())
		fields = append(fields, field.Name+":"+fieldKey)
	}

	structName := t.Name()
	if structName == "" {
		structName = "anonymous"
	}

	return "struct:" + structName + "{" + strings.Join(fields, ",") + "}"
}