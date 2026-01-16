package datasources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewSnapshotsDataSource(t *testing.T) {
	ds := NewSnapshotsDataSource()
	if ds == nil {
		t.Fatal("expected non-nil data source")
	}

	var _ datasource.DataSource = ds
	var _ datasource.DataSourceWithConfigure = ds.(*SnapshotsDataSource)
}

func TestSnapshotsDataSource_Metadata(t *testing.T) {
	ds := NewSnapshotsDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_snapshots" {
		t.Errorf("expected TypeName 'truenas_snapshots', got %q", resp.TypeName)
	}
}

func TestSnapshotsDataSource_Schema(t *testing.T) {
	ds := NewSnapshotsDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify dataset_id is required
	datasetIDAttr, ok := resp.Schema.Attributes["dataset_id"]
	if !ok {
		t.Fatal("expected 'dataset_id' attribute in schema")
	}
	if !datasetIDAttr.IsRequired() {
		t.Error("expected 'dataset_id' attribute to be required")
	}

	// Verify snapshots is computed
	snapshotsAttr, ok := resp.Schema.Attributes["snapshots"]
	if !ok {
		t.Fatal("expected 'snapshots' attribute in schema")
	}
	if !snapshotsAttr.IsComputed() {
		t.Error("expected 'snapshots' attribute to be computed")
	}
}

func TestSnapshotsDataSource_Configure_Success(t *testing.T) {
	ds := NewSnapshotsDataSource().(*SnapshotsDataSource)

	mockClient := &client.MockClient{}

	req := datasource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func getSnapshotsDataSourceSchema(t *testing.T) datasource.SchemaResponse {
	t.Helper()
	ds := NewSnapshotsDataSource()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

func TestSnapshotsDataSource_Read_Success(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[
					{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"holds": {},
						"properties": {
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					},
					{
						"id": "tank/data@snap2",
						"name": "snap2",
						"dataset": "tank/data",
						"holds": {"terraform": true},
						"properties": {
							"used": {"parsed": 512},
							"referenced": {"parsed": 1024}
						}
					}
				]`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Read_Empty(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(data.Snapshots))
	}
}
