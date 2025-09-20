package api

import (
	"fmt"
	"reflect"
	"strings"
)

// TemplateVariable represents a template variable with type information
type TemplateVariable struct {
	Key   string
	Value interface{}
	Type  string
}

// TemplateContext holds template data and processing logic
type TemplateContext struct {
	variables map[string]interface{}
	metadata  map[string]string
}

// NewTemplateContext creates a new template context
func NewTemplateContext() *TemplateContext {
	return &TemplateContext{
		variables: make(map[string]interface{}),
		metadata:  make(map[string]string),
	}
}

// Set sets a single template variable
func (tc *TemplateContext) Set(key string, value interface{}) *TemplateContext {
	tc.variables[key] = value
	return tc
}

// SetMultiple sets multiple template variables from a map
func (tc *TemplateContext) SetMultiple(variables map[string]interface{}) *TemplateContext {
	for k, v := range variables {
		tc.variables[k] = v
	}
	return tc
}

// SetFromKeyValue sets variables from alternating key-value pairs
func (tc *TemplateContext) SetFromKeyValue(keyValues ...interface{}) *TemplateContext {
	if len(keyValues)%2 != 0 {
		panic("SetFromKeyValue requires an even number of arguments")
	}

	for i := 0; i < len(keyValues); i += 2 {
		key := fmt.Sprintf("%v", keyValues[i])
		value := keyValues[i+1]
		tc.variables[key] = value
	}
	return tc
}

// SetFromStruct sets variables from struct fields using reflection
func (tc *TemplateContext) SetFromStruct(data interface{}) *TemplateContext {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return tc
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if !value.CanInterface() {
			continue
		}

		// Check if field should be ignored
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue // Skip fields with json:"-"
		}

		// Use json tag if available, otherwise use field name
		key := field.Name
		if jsonTag != "" {
			if comma := strings.Index(jsonTag, ","); comma != -1 {
				key = jsonTag[:comma]
			} else {
				key = jsonTag
			}
		}

		tc.variables[strings.ToLower(key)] = value.Interface()
	}

	return tc
}

// Get retrieves a template variable
func (tc *TemplateContext) Get(key string) (interface{}, bool) {
	value, exists := tc.variables[key]
	return value, exists
}

// GetString retrieves a template variable as string
func (tc *TemplateContext) GetString(key string) string {
	if value, exists := tc.variables[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// GetVariables returns all template variables
func (tc *TemplateContext) GetVariables() map[string]interface{} {
	// Return a copy to prevent external modification
	result := make(map[string]interface{})
	for k, v := range tc.variables {
		result[k] = v
	}
	return result
}

// GetVariableInfo returns detailed information about all variables
func (tc *TemplateContext) GetVariableInfo() []TemplateVariable {
	var vars []TemplateVariable
	for k, v := range tc.variables {
		vars = append(vars, TemplateVariable{
			Key:   k,
			Value: v,
			Type:  reflect.TypeOf(v).String(),
		})
	}
	return vars
}

// SetMetadata sets metadata for template processing
func (tc *TemplateContext) SetMetadata(key, value string) *TemplateContext {
	tc.metadata[key] = value
	return tc
}

// GetMetadata retrieves metadata
func (tc *TemplateContext) GetMetadata(key string) string {
	return tc.metadata[key]
}

// GetAllMetadata returns all metadata
func (tc *TemplateContext) GetAllMetadata() map[string]string {
	result := make(map[string]string)
	for k, v := range tc.metadata {
		result[k] = v
	}
	return result
}

// Clear removes all variables and metadata
func (tc *TemplateContext) Clear() *TemplateContext {
	tc.variables = make(map[string]interface{})
	tc.metadata = make(map[string]string)
	return tc
}

// HasVariable checks if a variable exists
func (tc *TemplateContext) HasVariable(key string) bool {
	_, exists := tc.variables[key]
	return exists
}

// RemoveVariable removes a specific variable
func (tc *TemplateContext) RemoveVariable(key string) *TemplateContext {
	delete(tc.variables, key)
	return tc
}

// Merge merges another template context into this one
func (tc *TemplateContext) Merge(other *TemplateContext) *TemplateContext {
	for k, v := range other.variables {
		tc.variables[k] = v
	}
	for k, v := range other.metadata {
		tc.metadata[k] = v
	}
	return tc
}

// TemplateProcessor handles template operations
type TemplateProcessor struct {
	templateName string
	context      *TemplateContext
}

// NewTemplateProcessor creates a new template processor
func NewTemplateProcessor() *TemplateProcessor {
	return &TemplateProcessor{
		context: NewTemplateContext(),
	}
}

// SetTemplate sets the template name
func (tp *TemplateProcessor) SetTemplate(name string) *TemplateProcessor {
	tp.templateName = name
	return tp
}

// GetTemplate returns the current template name
func (tp *TemplateProcessor) GetTemplate() string {
	return tp.templateName
}

// GetContext returns the template context
func (tp *TemplateProcessor) GetContext() *TemplateContext {
	return tp.context
}

// Var sets a template variable (fluent interface)
func (tp *TemplateProcessor) Var(key string, value interface{}) *TemplateProcessor {
	tp.context.Set(key, value)
	return tp
}

// Vars sets multiple template variables (fluent interface)
func (tp *TemplateProcessor) Vars(variables map[string]interface{}) *TemplateProcessor {
	tp.context.SetMultiple(variables)
	return tp
}

// VarsFromKeyValue sets variables from alternating key-value pairs (fluent interface)
func (tp *TemplateProcessor) VarsFromKeyValue(keyValues ...interface{}) *TemplateProcessor {
	tp.context.SetFromKeyValue(keyValues...)
	return tp
}

// VarsFromStruct sets variables from struct (fluent interface)
func (tp *TemplateProcessor) VarsFromStruct(data interface{}) *TemplateProcessor {
	tp.context.SetFromStruct(data)
	return tp
}

// Validate checks if the template processor is ready for use
func (tp *TemplateProcessor) Validate() error {
	if tp.templateName == "" {
		return fmt.Errorf("template name cannot be empty")
	}
	return nil
}

// IsEmpty checks if no template variables are set
func (tp *TemplateProcessor) IsEmpty() bool {
	return len(tp.context.variables) == 0
}

// Reset clears all template data
func (tp *TemplateProcessor) Reset() *TemplateProcessor {
	tp.templateName = ""
	tp.context.Clear()
	return tp
}