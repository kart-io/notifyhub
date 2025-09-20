package api

import (
	"testing"
)

func TestTemplateContext_BasicOperations(t *testing.T) {
	ctx := NewTemplateContext()

	// Test Set
	ctx.Set("key1", "value1")
	if value, exists := ctx.Get("key1"); !exists || value != "value1" {
		t.Errorf("Expected key1=value1, got %v (exists: %v)", value, exists)
	}

	// Test GetString
	if str := ctx.GetString("key1"); str != "value1" {
		t.Errorf("Expected string 'value1', got '%s'", str)
	}

	// Test non-existent key
	if str := ctx.GetString("nonexistent"); str != "" {
		t.Errorf("Expected empty string for non-existent key, got '%s'", str)
	}
}

func TestTemplateContext_SetMultiple(t *testing.T) {
	ctx := NewTemplateContext()

	variables := map[string]interface{}{
		"name":  "John",
		"age":   30,
		"admin": true,
	}

	ctx.SetMultiple(variables)

	if name := ctx.GetString("name"); name != "John" {
		t.Errorf("Expected name 'John', got '%s'", name)
	}

	if age, exists := ctx.Get("age"); !exists || age != 30 {
		t.Errorf("Expected age 30, got %v", age)
	}

	if admin, exists := ctx.Get("admin"); !exists || admin != true {
		t.Errorf("Expected admin true, got %v", admin)
	}
}

func TestTemplateContext_SetFromKeyValue(t *testing.T) {
	ctx := NewTemplateContext()

	ctx.SetFromKeyValue("key1", "value1", "key2", 42, "key3", true)

	if value := ctx.GetString("key1"); value != "value1" {
		t.Errorf("Expected key1='value1', got '%s'", value)
	}

	if value, exists := ctx.Get("key2"); !exists || value != 42 {
		t.Errorf("Expected key2=42, got %v", value)
	}

	if value, exists := ctx.Get("key3"); !exists || value != true {
		t.Errorf("Expected key3=true, got %v", value)
	}
}

func TestTemplateContext_SetFromKeyValue_Panic(t *testing.T) {
	ctx := NewTemplateContext()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for odd number of arguments")
		}
	}()

	ctx.SetFromKeyValue("key1", "value1", "key2") // odd number
}

func TestTemplateContext_SetFromStruct(t *testing.T) {
	ctx := NewTemplateContext()

	type TestStruct struct {
		Name     string `json:"name"`
		Age      int    `json:"age"`
		IsAdmin  bool   `json:"is_admin"`
		Internal string `json:"-"` // should be ignored
		private  string // should be ignored
	}

	data := TestStruct{
		Name:     "John",
		Age:      30,
		IsAdmin:  true,
		Internal: "ignored",
		private:  "private",
	}

	ctx.SetFromStruct(data)

	if name := ctx.GetString("name"); name != "John" {
		t.Errorf("Expected name 'John', got '%s'", name)
	}

	if age, exists := ctx.Get("age"); !exists || age != 30 {
		t.Errorf("Expected age 30, got %v", age)
	}

	if admin, exists := ctx.Get("is_admin"); !exists || admin != true {
		t.Errorf("Expected is_admin true, got %v", admin)
	}

	// Internal should be ignored due to json:"-"
	if _, exists := ctx.Get("internal"); exists {
		t.Error("Expected Internal field to be ignored")
	}

	// private should be ignored due to unexported
	if _, exists := ctx.Get("private"); exists {
		t.Error("Expected private field to be ignored")
	}
}

func TestTemplateContext_GetVariableInfo(t *testing.T) {
	ctx := NewTemplateContext()
	ctx.Set("string", "value")
	ctx.Set("int", 42)
	ctx.Set("bool", true)

	info := ctx.GetVariableInfo()
	if len(info) != 3 {
		t.Errorf("Expected 3 variables, got %d", len(info))
	}

	// Find string variable
	var stringVar *TemplateVariable
	for _, v := range info {
		if v.Key == "string" {
			stringVar = &v
			break
		}
	}

	if stringVar == nil {
		t.Error("Expected to find string variable")
	} else {
		if stringVar.Value != "value" {
			t.Errorf("Expected string value 'value', got %v", stringVar.Value)
		}
		if stringVar.Type != "string" {
			t.Errorf("Expected string type 'string', got '%s'", stringVar.Type)
		}
	}
}

func TestTemplateContext_Metadata(t *testing.T) {
	ctx := NewTemplateContext()

	ctx.SetMetadata("template_type", "email")
	ctx.SetMetadata("priority", "high")

	if value := ctx.GetMetadata("template_type"); value != "email" {
		t.Errorf("Expected template_type 'email', got '%s'", value)
	}

	metadata := ctx.GetAllMetadata()
	if len(metadata) != 2 {
		t.Errorf("Expected 2 metadata items, got %d", len(metadata))
	}
}

func TestTemplateContext_Operations(t *testing.T) {
	ctx := NewTemplateContext()
	ctx.Set("key1", "value1")
	ctx.Set("key2", "value2")

	// Test HasVariable
	if !ctx.HasVariable("key1") {
		t.Error("Expected HasVariable to return true for existing key")
	}

	if ctx.HasVariable("nonexistent") {
		t.Error("Expected HasVariable to return false for non-existent key")
	}

	// Test RemoveVariable
	ctx.RemoveVariable("key1")
	if ctx.HasVariable("key1") {
		t.Error("Expected key1 to be removed")
	}

	// Test Clear
	ctx.Clear()
	if ctx.HasVariable("key2") {
		t.Error("Expected all variables to be cleared")
	}
}

func TestTemplateContext_Merge(t *testing.T) {
	ctx1 := NewTemplateContext()
	ctx1.Set("key1", "value1")
	ctx1.SetMetadata("meta1", "metavalue1")

	ctx2 := NewTemplateContext()
	ctx2.Set("key2", "value2")
	ctx2.SetMetadata("meta2", "metavalue2")

	ctx1.Merge(ctx2)

	if !ctx1.HasVariable("key1") || !ctx1.HasVariable("key2") {
		t.Error("Expected both variables to exist after merge")
	}

	if ctx1.GetMetadata("meta1") != "metavalue1" || ctx1.GetMetadata("meta2") != "metavalue2" {
		t.Error("Expected both metadata items to exist after merge")
	}
}

func TestTemplateProcessor_BasicOperations(t *testing.T) {
	processor := NewTemplateProcessor()

	processor.SetTemplate("test-template")
	if template := processor.GetTemplate(); template != "test-template" {
		t.Errorf("Expected template 'test-template', got '%s'", template)
	}

	// Test fluent interface
	processor.Var("key1", "value1").
		Vars(map[string]interface{}{"key2": "value2"}).
		VarsFromKeyValue("key3", "value3")

	ctx := processor.GetContext()
	if !ctx.HasVariable("key1") || !ctx.HasVariable("key2") || !ctx.HasVariable("key3") {
		t.Error("Expected all variables to be set through fluent interface")
	}
}

func TestTemplateProcessor_VarsFromStruct(t *testing.T) {
	processor := NewTemplateProcessor()

	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	user := User{Name: "John", Email: "john@example.com", Age: 30}
	processor.VarsFromStruct(user)

	ctx := processor.GetContext()
	if name := ctx.GetString("name"); name != "John" {
		t.Errorf("Expected name 'John', got '%s'", name)
	}

	if email := ctx.GetString("email"); email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", email)
	}
}

func TestTemplateProcessor_Validate(t *testing.T) {
	processor := NewTemplateProcessor()

	// Should fail without template
	if err := processor.Validate(); err == nil {
		t.Error("Expected validation to fail without template name")
	}

	processor.SetTemplate("test-template")
	if err := processor.Validate(); err != nil {
		t.Errorf("Expected validation to pass with template name, got error: %v", err)
	}
}

func TestTemplateProcessor_IsEmpty(t *testing.T) {
	processor := NewTemplateProcessor()

	if !processor.IsEmpty() {
		t.Error("Expected processor to be empty initially")
	}

	processor.Var("key", "value")
	if processor.IsEmpty() {
		t.Error("Expected processor to not be empty after adding variable")
	}
}

func TestTemplateProcessor_Reset(t *testing.T) {
	processor := NewTemplateProcessor()
	processor.SetTemplate("test").Var("key", "value")

	processor.Reset()

	if processor.GetTemplate() != "" {
		t.Error("Expected template to be cleared after reset")
	}

	if !processor.IsEmpty() {
		t.Error("Expected processor to be empty after reset")
	}
}

func BenchmarkTemplateContext_Set(b *testing.B) {
	ctx := NewTemplateContext()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.Set("key", "value")
	}
}

func BenchmarkTemplateContext_SetFromStruct(b *testing.B) {
	ctx := NewTemplateContext()

	type TestStruct struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	data := TestStruct{Name: "John", Email: "john@example.com", Age: 30}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.SetFromStruct(data)
	}
}

func BenchmarkTemplateProcessor_FluentInterface(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		processor := NewTemplateProcessor()
		processor.SetTemplate("test").
			Var("key1", "value1").
			Var("key2", "value2").
			Var("key3", "value3")
	}
}