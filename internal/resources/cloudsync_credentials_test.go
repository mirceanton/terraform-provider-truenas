package resources

import (
	"context"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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

func TestCloudSyncCredentialsResource_Configure_Success(t *testing.T) {
	r := NewCloudSyncCredentialsResource().(*CloudSyncCredentialsResource)

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

func TestCloudSyncCredentialsResource_Configure_NilProviderData(t *testing.T) {
	r := NewCloudSyncCredentialsResource().(*CloudSyncCredentialsResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestCloudSyncCredentialsResource_Configure_WrongType(t *testing.T) {
	r := NewCloudSyncCredentialsResource().(*CloudSyncCredentialsResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// Test helpers

func getCloudSyncCredentialsResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewCloudSyncCredentialsResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

// cloudSyncCredentialsModelParams holds parameters for creating test model values.
type cloudSyncCredentialsModelParams struct {
	ID    interface{}
	Name  interface{}
	S3    *s3BlockParams
	B2    *b2BlockParams
	GCS   *gcsBlockParams
	Azure *azureBlockParams
}

type s3BlockParams struct {
	AccessKeyID     interface{}
	SecretAccessKey interface{}
	Endpoint        interface{}
	Region          interface{}
}

type b2BlockParams struct {
	Account interface{}
	Key     interface{}
}

type gcsBlockParams struct {
	ServiceAccountCredentials interface{}
}

type azureBlockParams struct {
	Account interface{}
	Key     interface{}
}

func createCloudSyncCredentialsModelValue(p cloudSyncCredentialsModelParams) tftypes.Value {
	s3Value := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"access_key_id":     tftypes.String,
			"secret_access_key": tftypes.String,
			"endpoint":          tftypes.String,
			"region":            tftypes.String,
		},
	}, nil)
	if p.S3 != nil {
		s3Value = tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"access_key_id":     tftypes.String,
				"secret_access_key": tftypes.String,
				"endpoint":          tftypes.String,
				"region":            tftypes.String,
			},
		}, map[string]tftypes.Value{
			"access_key_id":     tftypes.NewValue(tftypes.String, p.S3.AccessKeyID),
			"secret_access_key": tftypes.NewValue(tftypes.String, p.S3.SecretAccessKey),
			"endpoint":          tftypes.NewValue(tftypes.String, p.S3.Endpoint),
			"region":            tftypes.NewValue(tftypes.String, p.S3.Region),
		})
	}

	b2Value := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"account": tftypes.String,
			"key":     tftypes.String,
		},
	}, nil)
	if p.B2 != nil {
		b2Value = tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"account": tftypes.String,
				"key":     tftypes.String,
			},
		}, map[string]tftypes.Value{
			"account": tftypes.NewValue(tftypes.String, p.B2.Account),
			"key":     tftypes.NewValue(tftypes.String, p.B2.Key),
		})
	}

	gcsValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"service_account_credentials": tftypes.String,
		},
	}, nil)
	if p.GCS != nil {
		gcsValue = tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"service_account_credentials": tftypes.String,
			},
		}, map[string]tftypes.Value{
			"service_account_credentials": tftypes.NewValue(tftypes.String, p.GCS.ServiceAccountCredentials),
		})
	}

	azureValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"account": tftypes.String,
			"key":     tftypes.String,
		},
	}, nil)
	if p.Azure != nil {
		azureValue = tftypes.NewValue(tftypes.Object{
			AttributeTypes: map[string]tftypes.Type{
				"account": tftypes.String,
				"key":     tftypes.String,
			},
		}, map[string]tftypes.Value{
			"account": tftypes.NewValue(tftypes.String, p.Azure.Account),
			"key":     tftypes.NewValue(tftypes.String, p.Azure.Key),
		})
	}

	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":   tftypes.String,
			"name": tftypes.String,
			"s3": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"access_key_id":     tftypes.String,
					"secret_access_key": tftypes.String,
					"endpoint":          tftypes.String,
					"region":            tftypes.String,
				},
			},
			"b2": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"account": tftypes.String,
					"key":     tftypes.String,
				},
			},
			"gcs": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"service_account_credentials": tftypes.String,
				},
			},
			"azure": tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"account": tftypes.String,
					"key":     tftypes.String,
				},
			},
		},
	}, map[string]tftypes.Value{
		"id":    tftypes.NewValue(tftypes.String, p.ID),
		"name":  tftypes.NewValue(tftypes.String, p.Name),
		"s3":    s3Value,
		"b2":    b2Value,
		"gcs":   gcsValue,
		"azure": azureValue,
	})
}
