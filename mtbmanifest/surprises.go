package mtbmanifest

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
)

// AnyTag captures the Name and Inner Content of unknown elements
type AnyTag struct {
	XMLName xml.Name
	Body    string `xml:",innerxml"`
}

// Helper to print surprises
func (t AnyTag) String() string {
	return fmt.Sprintf("<%s>: %s", t.XMLName.Local, t.Body)
}

// ReportSurprises is your generic entry point.
// Pass ANY struct (root of your tree) to this function.
func ReportSurprises(data interface{}) {
	fmt.Println("🔍 Scanning for hidden XML data...")
	walk(reflect.ValueOf(data), []string{})
	fmt.Println("✅ Scan complete.")
}

// walk recursively inspects fields
func walk(v reflect.Value, path []string) {
	// 1. Unwrap Pointers and Interfaces
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	// 2. Handle Slices (Iterate over items)
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			// Update path to include index, e.g., Versions[0]
			itemPath := append(path, fmt.Sprintf("[%d]", i))
			walk(v.Index(i), itemPath)
		}
		return
	}

	// 3. Handle Structs (The meat of the logic)
	if v.Kind() == reflect.Struct {
		typ := v.Type()

		// A. Check for "Surprises" field (Tags)
		if f := v.FieldByName("Surprises"); f.IsValid() {
			if f.Len() > 0 {
				printSurprises(path, f)
			}
		}

		// B. Check for "LostAttrs" field (Attributes)
		if f := v.FieldByName("LostAttrs"); f.IsValid() {
			if f.Len() > 0 {
				printAttrs(path, f)
			}
		}

		// C. Recurse into all other fields to find children
		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i)
			fieldType := typ.Field(i)

			// Skip unexported fields (lowercase names)
			if fieldType.PkgPath != "" {
				continue
			}

			// Don't recurse into the surprise fields themselves
			if fieldType.Name == "Surprises" || fieldType.Name == "LostAttrs" {
				continue
			}

			// Optimization: Only recurse if it looks like a container (Slice, Struct, Ptr)
			k := fieldVal.Kind()
			if k == reflect.Struct || k == reflect.Slice || k == reflect.Ptr {
				// Append field name to path, e.g., "Versions"
				newPath := append(path, fieldType.Name)
				walk(fieldVal, newPath)
			}
		}
	}
}

// Helper to print unknown TAGS
func printSurprises(path []string, f reflect.Value) {
	// We assume f is []AnyTag
	for i := 0; i < f.Len(); i++ {
		tag := f.Index(i).Interface().(AnyTag)
		loc := strings.Join(path, ".")
		fmt.Printf("⚠️  Tag Surprise @ %s: <%s> %s\n", loc, tag.XMLName.Local, tag.Body)
	}
}

// Helper to print unknown ATTRIBUTES
func printAttrs(path []string, f reflect.Value) {
	// We assume f is []xml.Attr
	for i := 0; i < f.Len(); i++ {
		attr := f.Index(i).Interface().(xml.Attr)
		loc := strings.Join(path, ".")
		fmt.Printf("⚠️  Attr Surprise @ %s: %s=%q\n", loc, attr.Name.Local, attr.Value)
	}
}

// FindDeepSurprises returns a list of paths where unexpected JSON fields exist.
// data: The raw JSON bytes
// schema: A pointer to the struct you are mapping into (e.g., &Depender{})
func FindDeepSurprises(data []byte, schema interface{}) ([]string, error) {
	// 1. Unmarshal into the generic map (The "Truth" of what's actually there)
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	// 2. Get the Type of the schema struct
	// We check the Data (raw) against the Definition (val.Type())
	return inspect(raw, reflect.TypeOf(schema), ""), nil
}

func FindDeepSurprisesInStruct(data interface{}) []string {
	return inspect(data, reflect.TypeOf(data), "")
}

// inspect recursively compares the JSON value against the Go Type
func inspect(jsonVal interface{}, goType reflect.Type, path string) []string {
	var surprises []string

	// Handle nil/null
	if jsonVal == nil {
		return nil
	}

	// Unwrap Pointers in the Go definition
	if goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	// CASE A: JSON Object vs. Go Struct
	// We expect the JSON to be a map, and the Go type to be a Struct
	if mapVal, isMap := jsonVal.(map[string]interface{}); isMap {
		if goType.Kind() != reflect.Struct {
			// Mismatch: JSON is object, but Go expects something else (string, int, etc.)
			// We can flag this, or ignore it as a type error that Unmarshal would catch.
			return nil
		}

		// Iterate over every key actually present in the JSON
		for key, val := range mapVal {
			fieldName, fieldType, found := findFieldByJSONTag(goType, key)
			_ = fieldName // Not used here, but could be for more detailed reporting

			currentPath := key
			if path != "" {
				currentPath = path + "." + key
			}

			if !found {
				// 🚨 SURPRISE! This key is in JSON but not in our Struct
				surprises = append(surprises, fmt.Sprintf("%s (Value: %v)", currentPath, val))
			} else {
				// ✅ Known field. Recurse down!
				// We pass the inner JSON value and the inner Struct Field Type
				surprises = append(surprises, inspect(val, fieldType, currentPath)...)
			}
		}
		return surprises
	}

	// CASE B: JSON Array vs. Go Slice/Array
	if sliceVal, isSlice := jsonVal.([]interface{}); isSlice {
		if goType.Kind() != reflect.Slice && goType.Kind() != reflect.Array {
			return nil
		}

		// The type of the item INSIDE the slice
		elemType := goType.Elem()

		for i, item := range sliceVal {
			idxPath := fmt.Sprintf("%s[%d]", path, i)
			surprises = append(surprises, inspect(item, elemType, idxPath)...)
		}
		return surprises
	}

	// Case C: Primitives (String, Bool, Number)
	// We have reached a leaf node. No hidden fields here.
	return nil
}

// findFieldByJSONTag looks through the struct to find a matching JSON tag
func findFieldByJSONTag(t reflect.Type, jsonKey string) (string, reflect.Type, bool) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Get the tag
		tag := field.Tag.Get("json")

		// 1. Handle "json:name"
		if tag != "" {
			parts := strings.Split(tag, ",")
			tagName := parts[0]
			if tagName == "-" {
				continue
			} // explicitly ignored

			// JSON is typically case-insensitive, but let's be strict or loose as needed.
			// Using EqualFold allows "Start" to match "start"
			if strings.EqualFold(tagName, jsonKey) {
				return field.Name, field.Type, true
			}
		} else {
			// 2. Handle no tag (Go uses field name)
			if strings.EqualFold(field.Name, jsonKey) {
				return field.Name, field.Type, true
			}
		}
	}
	return "", nil, false
}
