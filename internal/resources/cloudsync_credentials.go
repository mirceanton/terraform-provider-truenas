package resources

import (
	"context"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.Resource = &CloudSyncCredentialsResource{}
var _ resource.ResourceWithConfigure = &CloudSyncCredentialsResource{}
var _ resource.ResourceWithImportState = &CloudSyncCredentialsResource{}

// CloudSyncCredentialsResource defines the resource implementation.
type CloudSyncCredentialsResource struct {
	client client.Client
}

// NewCloudSyncCredentialsResource creates a new CloudSyncCredentialsResource.
func NewCloudSyncCredentialsResource() resource.Resource {
	return &CloudSyncCredentialsResource{}
}

func (r *CloudSyncCredentialsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloudsync_credentials"
}

func (r *CloudSyncCredentialsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: implement
}

func (r *CloudSyncCredentialsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
