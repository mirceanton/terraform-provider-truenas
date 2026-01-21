package resources

import (
	"context"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewCronJobResource(t *testing.T) {
	r := NewCronJobResource()
	if r == nil {
		t.Fatal("NewCronJobResource returned nil")
	}

	_, ok := r.(*CronJobResource)
	if !ok {
		t.Fatalf("expected *CronJobResource, got %T", r)
	}

	// Verify interface implementations
	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*CronJobResource)
	var _ resource.ResourceWithImportState = r.(*CronJobResource)
}

func TestCronJobResource_Metadata(t *testing.T) {
	r := NewCronJobResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_cron_job" {
		t.Errorf("expected TypeName 'truenas_cron_job', got %q", resp.TypeName)
	}
}

func TestCronJobResource_Configure_Success(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)

	mockClient := &client.MockClient{}

	req := resource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if r.client == nil {
		t.Error("expected client to be set")
	}
}

func TestCronJobResource_Configure_NilProviderData(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestCronJobResource_Configure_WrongType(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestCronJobResource_Schema(t *testing.T) {
	r := NewCronJobResource()

	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := schemaResp.Schema.Attributes
	if attrs["id"] == nil {
		t.Error("expected 'id' attribute")
	}
	if attrs["user"] == nil {
		t.Error("expected 'user' attribute")
	}
	if attrs["command"] == nil {
		t.Error("expected 'command' attribute")
	}
	if attrs["description"] == nil {
		t.Error("expected 'description' attribute")
	}
	if attrs["enabled"] == nil {
		t.Error("expected 'enabled' attribute")
	}
	if attrs["stdout"] == nil {
		t.Error("expected 'stdout' attribute")
	}
	if attrs["stderr"] == nil {
		t.Error("expected 'stderr' attribute")
	}

	// Verify blocks exist
	blocks := schemaResp.Schema.Blocks
	if blocks["schedule"] == nil {
		t.Error("expected 'schedule' block")
	}
}
