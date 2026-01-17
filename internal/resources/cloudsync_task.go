package resources

import (
	"context"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &CloudSyncTaskResource{}
	_ resource.ResourceWithConfigure   = &CloudSyncTaskResource{}
	_ resource.ResourceWithImportState = &CloudSyncTaskResource{}
)

// CloudSyncTaskResourceModel describes the resource data model.
type CloudSyncTaskResourceModel struct {
	ID                 types.String     `tfsdk:"id"`
	Description        types.String     `tfsdk:"description"`
	Path               types.String     `tfsdk:"path"`
	Credentials        types.Int64      `tfsdk:"credentials"`
	Direction          types.String     `tfsdk:"direction"`
	TransferMode       types.String     `tfsdk:"transfer_mode"`
	Snapshot           types.Bool       `tfsdk:"snapshot"`
	Transfers          types.Int64      `tfsdk:"transfers"`
	BWLimit            types.String     `tfsdk:"bwlimit"`
	Exclude            types.List       `tfsdk:"exclude"`
	FollowSymlinks     types.Bool       `tfsdk:"follow_symlinks"`
	CreateEmptySrcDirs types.Bool       `tfsdk:"create_empty_src_dirs"`
	Enabled            types.Bool       `tfsdk:"enabled"`
	SyncOnChange       types.Bool       `tfsdk:"sync_on_change"`
	Schedule           *ScheduleBlock   `tfsdk:"schedule"`
	Encryption         *EncryptionBlock `tfsdk:"encryption"`
	S3                 *TaskS3Block     `tfsdk:"s3"`
	B2                 *TaskB2Block     `tfsdk:"b2"`
	GCS                *TaskGCSBlock    `tfsdk:"gcs"`
	Azure              *TaskAzureBlock  `tfsdk:"azure"`
}

// ScheduleBlock represents cron schedule settings.
type ScheduleBlock struct {
	Minute types.String `tfsdk:"minute"`
	Hour   types.String `tfsdk:"hour"`
	Dom    types.String `tfsdk:"dom"`
	Month  types.String `tfsdk:"month"`
	Dow    types.String `tfsdk:"dow"`
}

// EncryptionBlock represents encryption settings for cloud storage.
type EncryptionBlock struct {
	Password types.String `tfsdk:"password"`
	Salt     types.String `tfsdk:"salt"`
}

// TaskS3Block represents S3-compatible storage settings.
type TaskS3Block struct {
	Bucket types.String `tfsdk:"bucket"`
	Folder types.String `tfsdk:"folder"`
}

// TaskB2Block represents Backblaze B2 storage settings.
type TaskB2Block struct {
	Bucket types.String `tfsdk:"bucket"`
	Folder types.String `tfsdk:"folder"`
}

// TaskGCSBlock represents Google Cloud Storage settings.
type TaskGCSBlock struct {
	Bucket types.String `tfsdk:"bucket"`
	Folder types.String `tfsdk:"folder"`
}

// TaskAzureBlock represents Azure Blob Storage settings.
type TaskAzureBlock struct {
	Container types.String `tfsdk:"container"`
	Folder    types.String `tfsdk:"folder"`
}

// CloudSyncTaskResource defines the resource implementation.
type CloudSyncTaskResource struct {
	client client.Client
}

// NewCloudSyncTaskResource creates a new CloudSyncTaskResource.
func NewCloudSyncTaskResource() resource.Resource {
	return &CloudSyncTaskResource{}
}

func (r *CloudSyncTaskResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloudsync_task"
}

func (r *CloudSyncTaskResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages cloud sync backup tasks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Task ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Task description.",
				Required:    true,
			},
			"path": schema.StringAttribute{
				Description: "Local path to sync.",
				Required:    true,
			},
			"credentials": schema.Int64Attribute{
				Description: "Cloud sync credentials ID.",
				Required:    true,
			},
			"direction": schema.StringAttribute{
				Description: "Sync direction: push, pull, or sync.",
				Required:    true,
			},
			"transfer_mode": schema.StringAttribute{
				Description: "Transfer mode: sync, copy, or move.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("sync"),
			},
			"snapshot": schema.BoolAttribute{
				Description: "Take a snapshot before sync.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"transfers": schema.Int64Attribute{
				Description: "Number of simultaneous file transfers.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(4),
			},
			"bwlimit": schema.StringAttribute{
				Description: "Bandwidth limit in KB/s or schedule.",
				Optional:    true,
			},
			"exclude": schema.ListAttribute{
				Description: "Patterns to exclude from sync.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"follow_symlinks": schema.BoolAttribute{
				Description: "Follow symbolic links.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"create_empty_src_dirs": schema.BoolAttribute{
				Description: "Create empty source directories on destination.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enabled": schema.BoolAttribute{
				Description: "Enable the task.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"sync_on_change": schema.BoolAttribute{
				Description: "Fire-and-forget sync after create or update.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"schedule": schema.SingleNestedBlock{
				Description: "Cron schedule for the task.",
				Attributes: map[string]schema.Attribute{
					"minute": schema.StringAttribute{
						Description: "Minute (0-59 or cron expression).",
						Required:    true,
					},
					"hour": schema.StringAttribute{
						Description: "Hour (0-23 or cron expression).",
						Required:    true,
					},
					"dom": schema.StringAttribute{
						Description: "Day of month (1-31 or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
					"month": schema.StringAttribute{
						Description: "Month (1-12 or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
					"dow": schema.StringAttribute{
						Description: "Day of week (0-6 or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
				},
			},
			"encryption": schema.SingleNestedBlock{
				Description: "Encryption settings for cloud storage.",
				Attributes: map[string]schema.Attribute{
					"password": schema.StringAttribute{
						Description: "Encryption password.",
						Required:    true,
						Sensitive:   true,
					},
					"salt": schema.StringAttribute{
						Description: "Encryption salt.",
						Optional:    true,
						Computed:    true,
						Sensitive:   true,
					},
				},
			},
			"s3": schema.SingleNestedBlock{
				Description: "S3-compatible storage settings.",
				Attributes: map[string]schema.Attribute{
					"bucket": schema.StringAttribute{
						Description: "Bucket name.",
						Required:    true,
					},
					"folder": schema.StringAttribute{
						Description: "Folder path within the bucket.",
						Optional:    true,
					},
				},
			},
			"b2": schema.SingleNestedBlock{
				Description: "Backblaze B2 storage settings.",
				Attributes: map[string]schema.Attribute{
					"bucket": schema.StringAttribute{
						Description: "Bucket name.",
						Required:    true,
					},
					"folder": schema.StringAttribute{
						Description: "Folder path within the bucket.",
						Optional:    true,
					},
				},
			},
			"gcs": schema.SingleNestedBlock{
				Description: "Google Cloud Storage settings.",
				Attributes: map[string]schema.Attribute{
					"bucket": schema.StringAttribute{
						Description: "Bucket name.",
						Required:    true,
					},
					"folder": schema.StringAttribute{
						Description: "Folder path within the bucket.",
						Optional:    true,
					},
				},
			},
			"azure": schema.SingleNestedBlock{
				Description: "Azure Blob Storage settings.",
				Attributes: map[string]schema.Attribute{
					"container": schema.StringAttribute{
						Description: "Container name.",
						Required:    true,
					},
					"folder": schema.StringAttribute{
						Description: "Folder path within the container.",
						Optional:    true,
					},
				},
			},
		},
	}
}

func (r *CloudSyncTaskResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CloudSyncTaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: Implement
}

func (r *CloudSyncTaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: Implement
}

func (r *CloudSyncTaskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: Implement
}

func (r *CloudSyncTaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: Implement
}

func (r *CloudSyncTaskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
