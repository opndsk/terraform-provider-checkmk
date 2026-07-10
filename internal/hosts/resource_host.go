package hosts

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-checkmk/internal/client"
	"github.com/terraform-provider-checkmk/internal/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                   = &HostResource{}
	_ resource.ResourceWithImportState    = &HostResource{}
	_ resource.ResourceWithValidateConfig = &HostResource{}
)

func NewHostResource() resource.Resource {
	return &HostResource{}
}

// HostResource defines the resource implementation.
type HostResource struct {
	providerData *common.ProviderData
}

// HostResourceModel describes the resource data model.
type HostResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	HostName              types.String `tfsdk:"host_name"`
	Folder                types.String `tfsdk:"folder"`
	Attributes            types.Map    `tfsdk:"attributes"`
	Activate              types.String `tfsdk:"activate"`
	ForceForeignChanges   types.Bool   `tfsdk:"force_foreign_changes"`
	StrictResourceLocking types.Bool   `tfsdk:"strict_resource_locking"`
}

func (r *HostResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (r *HostResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CheckMK host configuration.",
		Attributes: map[string]schema.Attribute{
			"id":        common.ComputedIDAttribute("Host identifier (same as host_name)"),
			"host_name": common.RequiredIDAttribute("The hostname or IP address of the host. This serves as the unique identifier."),
			"folder": schema.StringAttribute{
				MarkdownDescription: "The folder path where the host will be created. Default is '/' (root folder). " +
					"Path delimiters can be `~`, `/`, or `\\`.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "Host attributes as key-value pairs. Common attributes include:\n" +
					"  - `alias`: Human-readable host alias\n" +
					"  - `ipaddress`: IP address of the host\n" +
					"  - `site`: CheckMK site ID\n" +
					"  - `tag_agent`: Agent type (e.g., 'cmk-agent', 'snmp-v2')\n\n" +
					"All attributes are replaced on update (full replacement strategy).",
				ElementType: types.StringType,
				Optional:    true,
			},
			"activate":                common.ActivateAttribute(),
			"force_foreign_changes":   common.ForceForeignChangesAttribute(),
			"strict_resource_locking": common.StrictResourceLockingAttribute(),
		},
	}
}

func (r *HostResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.providerData = common.ConfigureResource(req, resp)
}

func (r *HostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data HostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := common.BuildBaseConfig(r.providerData, data.Activate, data.ForceForeignChanges, data.StrictResourceLocking)

	// Convert attributes map to map[string]interface{}, promoting unprefixed
	// built-in tag groups (e.g. "agent" -> "tag_agent") to their API form.
	attributes := make(map[string]interface{})
	if !data.Attributes.IsNull() {
		promoter := common.NewAttributePromoter(r.providerData, "checkmk_host")
		for key, value := range data.Attributes.Elements() {
			if strValue, ok := value.(types.String); ok {
				attributes[promoter.APIKey(key)] = strValue.ValueString()
			}
		}
	}

	// Set default folder if not specified
	folder := "/"
	if !data.Folder.IsNull() {
		folder = data.Folder.ValueString()
	}

	// Create host via API
	createReq := &client.HostCreateRequest{
		HostName:   data.HostName.ValueString(),
		Folder:     folder,
		Attributes: attributes,
	}

	host, err := r.providerData.Client.CreateHost(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create host: %s", err))
		return
	}

	// Activate changes if configured
	if err := common.TrackAndActivate(ctx, r.providerData, cfg, "host"); err != nil {
		common.AddActivationWarning(resp, "Host", "created", err)
	}

	// Set computed values
	data.ID = types.StringValue(host.ID)
	if data.Folder.IsNull() {
		data.Folder = types.StringValue(host.Extensions.Folder)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host, err := r.providerData.Client.GetHost(ctx, data.HostName.ValueString())
	if err != nil {
		if common.HandleNotFoundOnRead(err, resp) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read host: %s", err))
		return
	}

	// Update state with API data
	data.ID = types.StringValue(host.ID)
	data.Folder = types.StringValue(host.Extensions.Folder)

	// Convert attributes from API response
	if len(host.Extensions.Attributes) > 0 {
		attrMap := make(map[string]string)

		if data.Attributes.IsNull() {
			// During import, we don't have previous state, so include all
			// attributes using the keys exactly as the API returns them.
			for key, value := range host.Extensions.Attributes {
				if strValue, ok := value.(string); ok {
					attrMap[key] = strValue
				}
			}
		} else {
			// Only include attributes that we're managing to avoid drift from
			// unmanaged attributes. Look each managed key up under its API form
			// (promoting unprefixed tag groups) but store it back under the key
			// the user configured, so "agent" stays "agent" in state.
			promoter := common.NewAttributePromoter(r.providerData, "checkmk_host")
			for key := range data.Attributes.Elements() {
				if value, exists := host.Extensions.Attributes[promoter.APIKey(key)]; exists {
					if strValue, ok := value.(string); ok {
						attrMap[key] = strValue
					}
				}
			}
		}

		var diags diag.Diagnostics
		data.Attributes, diags = types.MapValueFrom(ctx, types.StringType, attrMap)
		resp.Diagnostics.Append(diags...)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data HostResourceModel
	var state HostResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := common.BuildBaseConfig(r.providerData, data.Activate, data.ForceForeignChanges, data.StrictResourceLocking)
	currentFolder := common.GetStringValueOr(state.Folder, "/")
	desiredFolder := common.GetStringValueOr(data.Folder, currentFolder)
	folderChanged := currentFolder != desiredFolder
	attributesProvided := !data.Attributes.IsNull() && !data.Attributes.IsUnknown()
	attributesChanged := attributesProvided && !state.Attributes.Equal(data.Attributes)

	// Convert attributes map to map[string]interface{}, promoting unprefixed
	// built-in tag groups (e.g. "agent" -> "tag_agent") to their API form.
	var attributes map[string]interface{}
	if attributesProvided {
		attributes = make(map[string]interface{})
		promoter := common.NewAttributePromoter(r.providerData, "checkmk_host")
		for key, value := range data.Attributes.Elements() {
			if strValue, ok := value.(types.String); ok {
				attributes[promoter.APIKey(key)] = strValue.ValueString()
			}
		}
	}

	if !folderChanged && !attributesChanged {
		data.ID = types.StringValue(data.HostName.ValueString())
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	getETag := func(ctx context.Context) (string, error) {
		return common.FetchETagIfStrict(ctx, cfg.StrictResourceLocking, func(ctx context.Context) (string, error) {
			result, err := r.providerData.Client.GetHostWithETag(ctx, data.HostName.ValueString())
			if err != nil {
				return "", err
			}
			return result.ETag, nil
		})
	}

	var host *client.Host

	if folderChanged {
		etag, err := getETag(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fetch host ETag before move: %s", err))
			return
		}

		host, err = r.providerData.Client.MoveHost(ctx, data.HostName.ValueString(), desiredFolder, etag)
		if err != nil {
			if common.HandleDriftWarning(err, resp, data.HostName.ValueString()) {
				return
			}
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to move host: %s", err))
			return
		}
	}

	if attributesChanged {
		etag, err := getETag(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fetch host ETag before update: %s", err))
			return
		}

		updateReq := &client.HostUpdateRequest{
			Attributes: attributes,
		}

		host, err = r.providerData.Client.UpdateHost(ctx, data.HostName.ValueString(), updateReq, etag)
		if err != nil {
			if common.HandleDriftWarning(err, resp, data.HostName.ValueString()) {
				return
			}
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update host: %s", err))
			return
		}
	}

	// Activate changes if configured
	if err := common.TrackAndActivate(ctx, r.providerData, cfg, "host"); err != nil {
		common.AddActivationWarning(resp, "Host", "updated", err)
	}

	// Update state with API response
	data.ID = types.StringValue(data.HostName.ValueString())
	if host != nil {
		data.ID = types.StringValue(host.ID)
		data.Folder = types.StringValue(host.Extensions.Folder)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data HostResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := common.BuildBaseConfig(r.providerData, data.Activate, data.ForceForeignChanges, data.StrictResourceLocking)

	etag, err := common.FetchETagIfStrict(ctx, cfg.StrictResourceLocking, func(ctx context.Context) (string, error) {
		result, err := r.providerData.Client.GetHostWithETag(ctx, data.HostName.ValueString())
		if err != nil {
			if apiErr, ok := err.(*client.APIError); ok && apiErr.Status == 404 {
				return "", nil
			}
			return "", err
		}
		return result.ETag, nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to fetch host ETag: %s", err))
		return
	}

	if err := r.providerData.Client.DeleteHost(ctx, data.HostName.ValueString(), etag); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete host: %s", err))
		return
	}

	// Activate changes if configured
	if err := common.TrackAndActivate(ctx, r.providerData, cfg, "host"); err != nil {
		common.AddActivationWarning(resp, "Host", "deleted", err)
	}
}

func (r *HostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by host_name
	resource.ImportStatePassthroughID(ctx, path.Root("host_name"), req, resp)
}

// ValidateConfig validates the resource configuration before apply.
// This uses the generated types to check attribute names and enum values.
func (r *HostResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	// Provider data may not be available during early validation
	if r.providerData == nil {
		return
	}

	var data HostResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate host attributes using generated types
	validator := common.NewAttributeValidator(r.providerData)
	resp.Diagnostics.Append(validator.ValidateHostAttributes(ctx, data.Attributes, path.Root("attributes"))...)
}
