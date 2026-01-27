package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestCaseInsensitiveStringType_Equal(t *testing.T) {
	t.Parallel()

	t1 := CaseInsensitiveStringType{}
	t2 := CaseInsensitiveStringType{}
	t3 := basetypes.StringType{}

	if !t1.Equal(t2) {
		t.Error("expected CaseInsensitiveStringType to equal itself")
	}

	if t1.Equal(t3) {
		t.Error("expected CaseInsensitiveStringType to not equal StringType")
	}
}

func TestCaseInsensitiveStringType_String(t *testing.T) {
	t.Parallel()

	cisType := CaseInsensitiveStringType{}
	if cisType.String() != "CaseInsensitiveStringType" {
		t.Errorf("expected String() to return 'CaseInsensitiveStringType', got %q", cisType.String())
	}
}

func TestCaseInsensitiveStringType_ValueType(t *testing.T) {
	t.Parallel()

	cisType := CaseInsensitiveStringType{}
	val := cisType.ValueType(context.Background())

	if _, ok := val.(CaseInsensitiveStringValue); !ok {
		t.Errorf("expected ValueType to return CaseInsensitiveStringValue, got %T", val)
	}
}

func TestCaseInsensitiveStringType_ValueFromString(t *testing.T) {
	t.Parallel()

	cisType := CaseInsensitiveStringType{}
	stringVal := basetypes.NewStringValue("test")

	val, diags := cisType.ValueFromString(context.Background(), stringVal)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	cisVal, ok := val.(CaseInsensitiveStringValue)
	if !ok {
		t.Fatalf("expected CaseInsensitiveStringValue, got %T", val)
	}

	if cisVal.ValueString() != "test" {
		t.Errorf("expected 'test', got %q", cisVal.ValueString())
	}
}

func TestCaseInsensitiveStringType_ValueFromTerraform(t *testing.T) {
	t.Parallel()

	cisType := CaseInsensitiveStringType{}

	tests := []struct {
		name     string
		tfValue  tftypes.Value
		expected string
		isNull   bool
	}{
		{
			name:     "normal string",
			tfValue:  tftypes.NewValue(tftypes.String, "hello"),
			expected: "hello",
		},
		{
			name:    "null string",
			tfValue: tftypes.NewValue(tftypes.String, nil),
			isNull:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val, err := cisType.ValueFromTerraform(context.Background(), tc.tfValue)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cisVal, ok := val.(CaseInsensitiveStringValue)
			if !ok {
				t.Fatalf("expected CaseInsensitiveStringValue, got %T", val)
			}

			if tc.isNull {
				if !cisVal.IsNull() {
					t.Error("expected null value")
				}
			} else {
				if cisVal.ValueString() != tc.expected {
					t.Errorf("expected %q, got %q", tc.expected, cisVal.ValueString())
				}
			}
		})
	}
}

func TestCaseInsensitiveStringValue_Type(t *testing.T) {
	t.Parallel()

	cisVal := NewCaseInsensitiveStringValue("test")
	cisType := cisVal.Type(context.Background())

	if _, ok := cisType.(CaseInsensitiveStringType); !ok {
		t.Errorf("expected CaseInsensitiveStringType, got %T", cisType)
	}
}

func TestCaseInsensitiveStringValue_Equal(t *testing.T) {
	t.Parallel()

	v1 := NewCaseInsensitiveStringValue("test")
	v2 := NewCaseInsensitiveStringValue("test")
	v3 := NewCaseInsensitiveStringValue("TEST")
	v4 := basetypes.NewStringValue("test")

	// Equal checks exact value equality (not semantic equality)
	if !v1.Equal(v2) {
		t.Error("expected same values to be equal")
	}

	// Equal is NOT case-insensitive (that's what StringSemanticEquals is for)
	if v1.Equal(v3) {
		t.Error("Equal() should not be case-insensitive (StringSemanticEquals is)")
	}

	// Different types are not equal
	if v1.Equal(v4) {
		t.Error("expected different types to not be equal")
	}
}

func TestCaseInsensitiveStringValue_StringSemanticEquals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value1   CaseInsensitiveStringValue
		value2   basetypes.StringValuable
		expected bool
	}{
		{
			name:     "same case",
			value1:   NewCaseInsensitiveStringValue("RUNNING"),
			value2:   NewCaseInsensitiveStringValue("RUNNING"),
			expected: true,
		},
		{
			name:     "lowercase vs uppercase",
			value1:   NewCaseInsensitiveStringValue("running"),
			value2:   NewCaseInsensitiveStringValue("RUNNING"),
			expected: true,
		},
		{
			name:     "uppercase vs lowercase",
			value1:   NewCaseInsensitiveStringValue("STOPPED"),
			value2:   NewCaseInsensitiveStringValue("stopped"),
			expected: true,
		},
		{
			name:     "mixed case",
			value1:   NewCaseInsensitiveStringValue("Running"),
			value2:   NewCaseInsensitiveStringValue("rUNNING"),
			expected: true,
		},
		{
			name:     "different values",
			value1:   NewCaseInsensitiveStringValue("RUNNING"),
			value2:   NewCaseInsensitiveStringValue("STOPPED"),
			expected: false,
		},
		{
			name:     "both null",
			value1:   NewCaseInsensitiveStringNull(),
			value2:   NewCaseInsensitiveStringNull(),
			expected: true,
		},
		{
			name:     "one null",
			value1:   NewCaseInsensitiveStringValue("test"),
			value2:   NewCaseInsensitiveStringNull(),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, diags := tc.value1.StringSemanticEquals(context.Background(), tc.value2)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}

			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestCaseInsensitiveStringValue_StringSemanticEquals_WithUnknown(t *testing.T) {
	t.Parallel()

	v1 := NewCaseInsensitiveStringValue("test")
	v2 := NewCaseInsensitiveStringUnknown()

	result, diags := v1.StringSemanticEquals(context.Background(), v2)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if result {
		t.Error("expected unknown values to not be semantically equal")
	}
}

func TestNewCaseInsensitiveStringValue(t *testing.T) {
	t.Parallel()

	v := NewCaseInsensitiveStringValue("hello")
	if v.ValueString() != "hello" {
		t.Errorf("expected 'hello', got %q", v.ValueString())
	}
	if v.IsNull() {
		t.Error("expected non-null value")
	}
	if v.IsUnknown() {
		t.Error("expected known value")
	}
}

func TestNewCaseInsensitiveStringNull(t *testing.T) {
	t.Parallel()

	v := NewCaseInsensitiveStringNull()
	if !v.IsNull() {
		t.Error("expected null value")
	}
}

func TestNewCaseInsensitiveStringUnknown(t *testing.T) {
	t.Parallel()

	v := NewCaseInsensitiveStringUnknown()
	if !v.IsUnknown() {
		t.Error("expected unknown value")
	}
}

func TestNewCaseInsensitiveStringPointerValue(t *testing.T) {
	t.Parallel()

	str := "test"
	v := NewCaseInsensitiveStringPointerValue(&str)
	if v.ValueString() != "test" {
		t.Errorf("expected 'test', got %q", v.ValueString())
	}

	v2 := NewCaseInsensitiveStringPointerValue(nil)
	if !v2.IsNull() {
		t.Error("expected null value for nil pointer")
	}
}

func TestCaseInsensitiveStringType_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ attr.Type = CaseInsensitiveStringType{}
	var _ basetypes.StringTypable = CaseInsensitiveStringType{}
}

func TestCaseInsensitiveStringValue_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ attr.Value = CaseInsensitiveStringValue{}
	var _ basetypes.StringValuable = CaseInsensitiveStringValue{}
	var _ basetypes.StringValuableWithSemanticEquals = CaseInsensitiveStringValue{}
}
