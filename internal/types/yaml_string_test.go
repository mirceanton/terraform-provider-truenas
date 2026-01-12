package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestYAMLStringValue_SemanticEquals_IdenticalStrings(t *testing.T) {
	ctx := context.Background()
	yaml := "services:\n  web:\n    image: nginx"

	v1 := NewYAMLStringValue(yaml)
	v2 := NewYAMLStringValue(yaml)

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if !equal {
		t.Error("expected identical YAML strings to be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_DifferentKeyOrder(t *testing.T) {
	ctx := context.Background()
	yaml1 := "services:\n  web:\n    image: nginx\n    ports:\n      - 80:80"
	yaml2 := "services:\n  web:\n    ports:\n      - 80:80\n    image: nginx"

	v1 := NewYAMLStringValue(yaml1)
	v2 := NewYAMLStringValue(yaml2)

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if !equal {
		t.Error("expected YAML with different key order to be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_DifferentWhitespace(t *testing.T) {
	ctx := context.Background()
	yaml1 := "services:\n  web:\n    image: nginx"
	yaml2 := "services:\n  web:\n    image:   nginx\n"

	v1 := NewYAMLStringValue(yaml1)
	v2 := NewYAMLStringValue(yaml2)

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if !equal {
		t.Error("expected YAML with different whitespace to be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_DifferentValues(t *testing.T) {
	ctx := context.Background()
	yaml1 := "services:\n  web:\n    image: nginx"
	yaml2 := "services:\n  web:\n    image: apache"

	v1 := NewYAMLStringValue(yaml1)
	v2 := NewYAMLStringValue(yaml2)

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if equal {
		t.Error("expected YAML with different values to NOT be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_NullValues(t *testing.T) {
	ctx := context.Background()

	v1 := NewYAMLStringNull()
	v2 := NewYAMLStringNull()

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if !equal {
		t.Error("expected null values to be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_NullVsNonNull(t *testing.T) {
	ctx := context.Background()

	v1 := NewYAMLStringNull()
	v2 := NewYAMLStringValue("services:\n  web:\n    image: nginx")

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if equal {
		t.Error("expected null and non-null to NOT be semantically equal")
	}
}

func TestYAMLStringValue_SemanticEquals_InvalidYAML(t *testing.T) {
	ctx := context.Background()
	validYAML := "services:\n  web:\n    image: nginx"
	invalidYAML := "services:\n  web:\n    image: nginx\n  invalid: [unclosed"

	v1 := NewYAMLStringValue(validYAML)
	v2 := NewYAMLStringValue(invalidYAML)

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	// Invalid YAML should fall back to string comparison
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if equal {
		t.Error("expected invalid YAML to fall back to string comparison (not equal)")
	}
}

func TestYAMLStringValue_Type(t *testing.T) {
	v := NewYAMLStringValue("test")
	typ := v.Type(context.Background())

	if _, ok := typ.(YAMLStringType); !ok {
		t.Errorf("expected YAMLStringType, got %T", typ)
	}
}

func TestYAMLStringType_ValueFromString(t *testing.T) {
	ctx := context.Background()
	typ := YAMLStringType{}

	stringValue := basetypes.NewStringValue("services:\n  web:\n    image: nginx")
	result, diags := typ.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}

	yamlValue, ok := result.(YAMLStringValue)
	if !ok {
		t.Fatalf("expected YAMLStringValue, got %T", result)
	}

	if yamlValue.ValueString() != "services:\n  web:\n    image: nginx" {
		t.Errorf("unexpected value: %q", yamlValue.ValueString())
	}
}

func TestYAMLStringType_Equal(t *testing.T) {
	typ1 := YAMLStringType{}
	typ2 := YAMLStringType{}

	// Same type should be equal
	if !typ1.Equal(typ2) {
		t.Error("expected YAMLStringType to be equal to another YAMLStringType")
	}

	// Different type should not be equal
	if typ1.Equal(basetypes.StringType{}) {
		t.Error("expected YAMLStringType to not be equal to StringType")
	}
}

func TestYAMLStringType_String(t *testing.T) {
	typ := YAMLStringType{}

	if typ.String() != "YAMLStringType" {
		t.Errorf("expected 'YAMLStringType', got %q", typ.String())
	}
}

func TestYAMLStringType_ValueType(t *testing.T) {
	typ := YAMLStringType{}
	ctx := context.Background()

	value := typ.ValueType(ctx)

	if _, ok := value.(YAMLStringValue); !ok {
		t.Errorf("expected YAMLStringValue, got %T", value)
	}
}

func TestYAMLStringType_ValueFromTerraform(t *testing.T) {
	ctx := context.Background()
	typ := YAMLStringType{}

	// Test with valid string value
	tfValue := tftypes.NewValue(tftypes.String, "services:\n  web:\n    image: nginx")
	result, err := typ.ValueFromTerraform(ctx, tfValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	yamlValue, ok := result.(YAMLStringValue)
	if !ok {
		t.Fatalf("expected YAMLStringValue, got %T", result)
	}

	if yamlValue.ValueString() != "services:\n  web:\n    image: nginx" {
		t.Errorf("unexpected value: %q", yamlValue.ValueString())
	}

	// Test with null value
	nullTfValue := tftypes.NewValue(tftypes.String, nil)
	nullResult, err := typ.ValueFromTerraform(ctx, nullTfValue)
	if err != nil {
		t.Fatalf("unexpected error for null: %v", err)
	}

	nullYamlValue, ok := nullResult.(YAMLStringValue)
	if !ok {
		t.Fatalf("expected YAMLStringValue for null, got %T", nullResult)
	}

	if !nullYamlValue.IsNull() {
		t.Error("expected null value")
	}

	// Test with unknown value
	unknownTfValue := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	unknownResult, err := typ.ValueFromTerraform(ctx, unknownTfValue)
	if err != nil {
		t.Fatalf("unexpected error for unknown: %v", err)
	}

	unknownYamlValue, ok := unknownResult.(YAMLStringValue)
	if !ok {
		t.Fatalf("expected YAMLStringValue for unknown, got %T", unknownResult)
	}

	if !unknownYamlValue.IsUnknown() {
		t.Error("expected unknown value")
	}
}

func TestYAMLStringValue_Equal(t *testing.T) {
	v1 := NewYAMLStringValue("test")
	v2 := NewYAMLStringValue("test")
	v3 := NewYAMLStringValue("different")

	// Same value should be equal
	if !v1.Equal(v2) {
		t.Error("expected equal values to be equal")
	}

	// Different value should not be equal
	if v1.Equal(v3) {
		t.Error("expected different values to not be equal")
	}

	// Different type should not be equal
	if v1.Equal(basetypes.NewStringValue("test")) {
		t.Error("expected YAMLStringValue to not be equal to StringValue")
	}
}

func TestNewYAMLStringUnknown(t *testing.T) {
	v := NewYAMLStringUnknown()

	if !v.IsUnknown() {
		t.Error("expected unknown value")
	}

	if v.IsNull() {
		t.Error("unknown value should not be null")
	}
}

func TestNewYAMLStringPointerValue(t *testing.T) {
	// Test with non-nil pointer
	s := "test value"
	v := NewYAMLStringPointerValue(&s)

	if v.IsNull() {
		t.Error("expected non-null value")
	}

	if v.ValueString() != "test value" {
		t.Errorf("expected 'test value', got %q", v.ValueString())
	}

	// Test with nil pointer
	vNil := NewYAMLStringPointerValue(nil)

	if !vNil.IsNull() {
		t.Error("expected null value for nil pointer")
	}
}

func TestYAMLStringValue_SemanticEquals_UnknownValues(t *testing.T) {
	ctx := context.Background()

	v1 := NewYAMLStringUnknown()
	v2 := NewYAMLStringValue("test")

	equal, diags := v1.StringSemanticEquals(ctx, v2)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}

	// Unknown vs known should not be equal
	if equal {
		t.Error("expected unknown and known values to not be semantically equal")
	}
}
