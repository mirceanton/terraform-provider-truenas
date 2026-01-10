package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewFileResource(t *testing.T) {
	r := NewFileResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	// Verify it implements the required interfaces
	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*FileResource)
	var _ resource.ResourceWithImportState = r.(*FileResource)
	var _ resource.ResourceWithValidateConfig = r.(*FileResource)
}

func TestFileResource_Metadata(t *testing.T) {
	r := NewFileResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_file" {
		t.Errorf("expected TypeName 'truenas_file', got %q", resp.TypeName)
	}
}
