package changelog

import (
	"fmt"
	"reflect"
)

// ComputeDiff compares two structs and returns a list of field changes.
// Both old and new must be the same type. Uses JSON tag names as field names.
// Skips fields without json tags and fields tagged with "-".
func ComputeDiff(old, new any) []FieldChange {
	if old == nil || new == nil {
		return nil
	}

	oldVal := reflect.ValueOf(old)
	newVal := reflect.ValueOf(new)

	// Dereference pointers.
	if oldVal.Kind() == reflect.Ptr {
		oldVal = oldVal.Elem()
	}
	if newVal.Kind() == reflect.Ptr {
		newVal = newVal.Elem()
	}

	if oldVal.Type() != newVal.Type() {
		return nil
	}

	if oldVal.Kind() != reflect.Struct {
		return nil
	}

	var changes []FieldChange
	t := oldVal.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		name, ok := diffFieldName(field)
		if !ok || isSystemField(name) {
			continue
		}

		oldField := oldVal.Field(i)
		newField := newVal.Field(i)

		if !reflect.DeepEqual(oldField.Interface(), newField.Interface()) {
			changes = append(changes, FieldChange{
				Field:    name,
				OldValue: formatValue(oldField),
				NewValue: formatValue(newField),
			})
		}
	}

	return changes
}

func diffFieldName(field reflect.StructField) (string, bool) {
	if !field.IsExported() {
		return "", false
	}

	jsonTag := field.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return "", false
	}

	for i, char := range jsonTag {
		if char == ',' {
			return jsonTag[:i], true
		}
	}

	return jsonTag, true
}

func isSystemField(name string) bool {
	switch name {
	case "id", "createdAt", "updatedAt", "createdBy", "updatedBy":
		return true
	default:
		return false
	}
}

func formatValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	// Handle nil pointers and interfaces.
	if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
		return nil
	}
	// Handle nil slices and maps.
	if (v.Kind() == reflect.Slice || v.Kind() == reflect.Map) && v.IsNil() {
		return nil
	}
	return fmt.Sprintf("%v", v.Interface())
}
