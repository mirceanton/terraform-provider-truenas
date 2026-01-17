package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewCloudSyncCredentialsResource(t *testing.T) {
	r := NewCloudSyncCredentialsResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*CloudSyncCredentialsResource)
	var _ resource.ResourceWithImportState = r.(*CloudSyncCredentialsResource)
}

func TestCloudSyncCredentialsResource_Metadata(t *testing.T) {
	r := NewCloudSyncCredentialsResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_cloudsync_credentials" {
		t.Errorf("expected TypeName 'truenas_cloudsync_credentials', got %q", resp.TypeName)
	}
}

func TestCloudSyncCredentialsResource_Schema(t *testing.T) {
	r := NewCloudSyncCredentialsResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify id attribute exists and is computed
	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute in schema")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' attribute to be computed")
	}

	// Verify name attribute exists and is required
	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if !nameAttr.IsRequired() {
		t.Error("expected 'name' attribute to be required")
	}

	// Verify provider blocks exist
	for _, block := range []string{"s3", "b2", "gcs", "azure"} {
		_, ok := resp.Schema.Blocks[block]
		if !ok {
			t.Errorf("expected '%s' block in schema", block)
		}
	}
}
