package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &CronJobResource{}
	_ resource.ResourceWithConfigure   = &CronJobResource{}
	_ resource.ResourceWithImportState = &CronJobResource{}
)

// CronJobResourceModel describes the resource data model.
type CronJobResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	User        types.String   `tfsdk:"user"`
	Command     types.String   `tfsdk:"command"`
	Description types.String   `tfsdk:"description"`
	Enabled     types.Bool     `tfsdk:"enabled"`
	Stdout      types.Bool     `tfsdk:"stdout"`
	Stderr      types.Bool     `tfsdk:"stderr"`
	Schedule    *ScheduleBlock `tfsdk:"schedule"`
}

// CronJobResource defines the resource implementation.
type CronJobResource struct {
	client client.Client
}

// NewCronJobResource creates a new CronJobResource.
func NewCronJobResource() resource.Resource {
	return &CronJobResource{}
}

func (r *CronJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cron_job"
}

func (r *CronJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages cron jobs for scheduled task execution.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Cron job ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user": schema.StringAttribute{
				Description: "User to run the command as.",
				Required:    true,
			},
			"command": schema.StringAttribute{
				Description: "Command to execute.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Job description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"enabled": schema.BoolAttribute{
				Description: "Enable the cron job.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"stdout": schema.BoolAttribute{
				Description: "Redirect stdout to syslog.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"stderr": schema.BoolAttribute{
				Description: "Redirect stderr to syslog.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
		Blocks: map[string]schema.Block{
			"schedule": schema.SingleNestedBlock{
				Description: "Cron schedule for the job.",
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
		},
	}
}

func (r *CronJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CronJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// queryCronJob queries a cron job by ID and returns the response.
func (r *CronJobResource) queryCronJob(ctx context.Context, id int64) (*api.CronJobResponse, error) {
	filter := [][]any{{"id", "=", id}}
	result, err := r.client.Call(ctx, "cronjob.query", filter)
	if err != nil {
		return nil, err
	}

	var jobs []api.CronJobResponse
	if err := json.Unmarshal(result, &jobs); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(jobs) == 0 {
		return nil, nil
	}

	return &jobs[0], nil
}

// buildCronJobParams builds the API params from the resource model.
func buildCronJobParams(data *CronJobResourceModel) map[string]any {
	params := map[string]any{
		"user":        data.User.ValueString(),
		"command":     data.Command.ValueString(),
		"description": data.Description.ValueString(),
		"enabled":     data.Enabled.ValueBool(),
		"stdout":      data.Stdout.ValueBool(),
		"stderr":      data.Stderr.ValueBool(),
	}

	if data.Schedule != nil {
		params["schedule"] = map[string]any{
			"minute": data.Schedule.Minute.ValueString(),
			"hour":   data.Schedule.Hour.ValueString(),
			"dom":    data.Schedule.Dom.ValueString(),
			"month":  data.Schedule.Month.ValueString(),
			"dow":    data.Schedule.Dow.ValueString(),
		}
	}

	return params
}

// mapCronJobToModel maps an API response to the resource model.
func mapCronJobToModel(job *api.CronJobResponse, data *CronJobResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(job.ID, 10))
	data.User = types.StringValue(job.User)
	data.Command = types.StringValue(job.Command)
	data.Description = types.StringValue(job.Description)
	data.Enabled = types.BoolValue(job.Enabled)
	data.Stdout = types.BoolValue(job.Stdout)
	data.Stderr = types.BoolValue(job.Stderr)

	if data.Schedule != nil {
		data.Schedule.Minute = types.StringValue(job.Schedule.Minute)
		data.Schedule.Hour = types.StringValue(job.Schedule.Hour)
		data.Schedule.Dom = types.StringValue(job.Schedule.Dom)
		data.Schedule.Month = types.StringValue(job.Schedule.Month)
		data.Schedule.Dow = types.StringValue(job.Schedule.Dow)
	}
}
