package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SnapshotResource{}
var _ resource.ResourceWithConfigure = &SnapshotResource{}
var _ resource.ResourceWithImportState = &SnapshotResource{}

// SnapshotResource defines the resource implementation.
type SnapshotResource struct {
	client client.Client
}

// SnapshotResourceModel describes the resource data model.
type SnapshotResourceModel struct {
	ID              types.String `tfsdk:"id"`
	DatasetID       types.String `tfsdk:"dataset_id"`
	Name            types.String `tfsdk:"name"`
	Hold            types.Bool   `tfsdk:"hold"`
	Recursive       types.Bool   `tfsdk:"recursive"`
	CreateTXG       types.String `tfsdk:"createtxg"`
	UsedBytes       types.Int64  `tfsdk:"used_bytes"`
	ReferencedBytes types.Int64  `tfsdk:"referenced_bytes"`
}

// NewSnapshotResource creates a new SnapshotResource.
func NewSnapshotResource() resource.Resource {
	return &SnapshotResource{}
}

func (r *SnapshotResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

func (r *SnapshotResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ZFS snapshot. Use for pre-upgrade backups and point-in-time recovery.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Snapshot identifier (dataset@name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dataset_id": schema.StringAttribute{
				Description: "Dataset ID to snapshot. Reference a truenas_dataset resource or data source.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Snapshot name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hold": schema.BoolAttribute{
				Description: "Prevent automatic deletion. Default: false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"recursive": schema.BoolAttribute{
				Description: "Include child datasets. Default: false. Only used at create time.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"createtxg": schema.StringAttribute{
				Description: "Transaction group when snapshot was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"used_bytes": schema.Int64Attribute{
				Description: "Space consumed by snapshot.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"referenced_bytes": schema.Int64Attribute{
				Description: "Space referenced by snapshot.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SnapshotResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *SnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Create not yet implemented")
}

func (r *SnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Read not yet implemented")
}

func (r *SnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Update not yet implemented")
}

func (r *SnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Delete not yet implemented")
}

func (r *SnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
