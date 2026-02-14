package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewZvolResource(t *testing.T) {
	r := NewZvolResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(r.(*ZvolResource))
	_ = resource.ResourceWithImportState(r.(*ZvolResource))
}

func TestZvolResource_Metadata(t *testing.T) {
	r := NewZvolResource().(*ZvolResource)
	req := resource.MetadataRequest{ProviderTypeName: "truenas"}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_zvol" {
		t.Errorf("expected type name 'truenas_zvol', got %q", resp.TypeName)
	}
}

func TestZvolResource_Schema(t *testing.T) {
	schemaResp := getZvolResourceSchema(t)

	expectedAttrs := []string{
		"id", "pool", "path", "parent",
		"volsize", "volblocksize", "sparse", "force_size",
		"compression", "comments",
		"force_destroy",
	}

	for _, attr := range expectedAttrs {
		if _, ok := schemaResp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing expected attribute %q", attr)
		}
	}
}

func TestZvolResource_Create_Basic(t *testing.T) {
	var createCalled bool
	var createParams map[string]any

	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.dataset.create" {
					createCalled = true
					createParams = params.(map[string]any)
					return json.RawMessage(`{"id":"tank/myvol"}`), nil
				}
				if method == "pool.dataset.query" {
					return json.RawMessage(mockZvolQueryResponse("tank/myvol", "lz4", "", 10737418240, "16K", false)), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	planValue := createZvolModelValue(defaultZvolPlanParams())

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if !createCalled {
		t.Fatal("expected pool.dataset.create to be called")
	}
	if createParams["name"] != "tank/myvol" {
		t.Errorf("expected name 'tank/myvol', got %v", createParams["name"])
	}
	if createParams["type"] != "VOLUME" {
		t.Errorf("expected type 'VOLUME', got %v", createParams["type"])
	}
	if createParams["volsize"] != int64(10737418240) {
		t.Errorf("expected volsize 10737418240, got %v", createParams["volsize"])
	}

	var model ZvolResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.ID.ValueString() != "tank/myvol" {
		t.Errorf("expected ID 'tank/myvol', got %q", model.ID.ValueString())
	}
}

func TestZvolResource_Create_WithOptionalFields(t *testing.T) {
	var createParams map[string]any

	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.dataset.create" {
					createParams = params.(map[string]any)
					return json.RawMessage(`{"id":"tank/myvol"}`), nil
				}
				if method == "pool.dataset.query" {
					return json.RawMessage(mockZvolQueryResponse("tank/myvol", "zstd", "test vol", 10737418240, "64K", true)), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.Volblocksize = strPtr("64K")
	p.Sparse = boolPtr(true)
	p.Compression = strPtr("zstd")
	p.Comments = strPtr("test vol")
	planValue := createZvolModelValue(p)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if createParams["volblocksize"] != "64K" {
		t.Errorf("expected volblocksize '64K', got %v", createParams["volblocksize"])
	}
	if createParams["sparse"] != true {
		t.Errorf("expected sparse true, got %v", createParams["sparse"])
	}
	if createParams["compression"] != "zstd" {
		t.Errorf("expected compression 'zstd', got %v", createParams["compression"])
	}
	if createParams["comments"] != "test vol" {
		t.Errorf("expected comments 'test vol', got %v", createParams["comments"])
	}
}

func TestZvolResource_Create_InvalidName(t *testing.T) {
	r := &ZvolResource{client: &client.MockClient{}}

	schemaResp := getZvolResourceSchema(t)
	p := zvolModelParams{Volsize: strPtr("10G")} // no pool/path
	planValue := createZvolModelValue(p)

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid name")
	}
}

func TestZvolResource_Create_APIError(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("pool not found")
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	planValue := createZvolModelValue(defaultZvolPlanParams())

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API failure")
	}
}

func TestZvolResource_Read_Basic(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(mockZvolQueryResponse("tank/myvol", "lz4", "", 10737418240, "16K", false)), nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/myvol")
	stateValue := createZvolModelValue(p)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model ZvolResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}
	if model.ID.ValueString() != "tank/myvol" {
		t.Errorf("expected ID 'tank/myvol', got %q", model.ID.ValueString())
	}
	if model.Compression.ValueString() != "lz4" {
		t.Errorf("expected compression 'lz4', got %q", model.Compression.ValueString())
	}
}

func TestZvolResource_Read_NotFound(t *testing.T) {
	r := &ZvolResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getZvolResourceSchema(t)
	p := defaultZvolPlanParams()
	p.ID = strPtr("tank/deleted")
	stateValue := createZvolModelValue(p)

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be removed (resource deleted outside Terraform)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed for deleted zvol")
	}
}

// -- Test helpers --

func getZvolResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewZvolResource().(*ZvolResource)
	resp := resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema error: %v", resp.Diagnostics)
	}
	return resp
}

// zvolObjectType returns the tftypes.Object for constructing test values.
func zvolObjectType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"volsize":       tftypes.String,
			"volblocksize":  tftypes.String,
			"sparse":        tftypes.Bool,
			"force_size":    tftypes.Bool,
			"compression":   tftypes.String,
			"comments":      tftypes.String,
			"force_destroy": tftypes.Bool,
		},
	}
}

type zvolModelParams struct {
	ID           *string
	Pool         *string
	Path         *string
	Parent       *string
	Volsize      *string
	Volblocksize *string
	Sparse       *bool
	ForceSize    *bool
	Compression  *string
	Comments     *string
	ForceDestroy *bool
}

func createZvolModelValue(p zvolModelParams) tftypes.Value {
	strVal := func(s *string) tftypes.Value {
		if s == nil {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, *s)
	}
	boolVal := func(b *bool) tftypes.Value {
		if b == nil {
			return tftypes.NewValue(tftypes.Bool, nil)
		}
		return tftypes.NewValue(tftypes.Bool, *b)
	}

	return tftypes.NewValue(zvolObjectType(), map[string]tftypes.Value{
		"id":            strVal(p.ID),
		"pool":          strVal(p.Pool),
		"path":          strVal(p.Path),
		"parent":        strVal(p.Parent),
		"volsize":       strVal(p.Volsize),
		"volblocksize":  strVal(p.Volblocksize),
		"sparse":        boolVal(p.Sparse),
		"force_size":    boolVal(p.ForceSize),
		"compression":   strVal(p.Compression),
		"comments":      strVal(p.Comments),
		"force_destroy": boolVal(p.ForceDestroy),
	})
}

func boolPtr(b bool) *bool { return &b }

func defaultZvolPlanParams() zvolModelParams {
	return zvolModelParams{
		Pool:    strPtr("tank"),
		Path:    strPtr("myvol"),
		Volsize: strPtr("10737418240"),
	}
}

// mockZvolQueryResponse returns a mock pool.dataset.query response for a zvol.
func mockZvolQueryResponse(id, compression, comments string, volsize int64, volblocksize string, sparse bool) string {
	return fmt.Sprintf(`[{
		"id": %q,
		"type": "VOLUME",
		"name": %q,
		"pool": "tank",
		"volsize": {"value": "%d", "parsed": %d},
		"volblocksize": {"value": %q, "parsed": 0},
		"sparse": {"value": "%t", "parsed": %t},
		"compression": {"value": %q},
		"comments": {"value": %q}
	}]`, id, id, volsize, volsize, volblocksize, sparse, sparse, compression, comments)
}
