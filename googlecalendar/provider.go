// Package googlecalendar manages Google calendar events with Terraform.
package googlecalendar

import (
	"context"
	"fmt"
	"runtime"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Ensure the implementation satisfies the provider.Provider interface.
var _ provider.Provider = &googleCalendarProvider{}

// googleCalendarProvider is the provider implementation.
type googleCalendarProvider struct {
	version string
}

// googleCalendarProviderModel describes the provider data model.
type googleCalendarProviderModel struct {
	Credentials types.String `tfsdk:"credentials"`
}

// New creates a new provider instance.
func New() provider.Provider {
	return &googleCalendarProvider{}
}

// Metadata returns the provider type name.
func (p *googleCalendarProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "googlecalendar"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *googleCalendarProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Google Calendar events.",
		Attributes: map[string]schema.Attribute{
			"credentials": schema.StringAttribute{
				Description: "Google Cloud credentials JSON. Can also be set via GOOGLE_CREDENTIALS, GOOGLE_CLOUD_KEYFILE_JSON, or GCLOUD_KEYFILE_JSON environment variables.",
				Optional:    true,
			},
		},
	}
}

// Configure prepares a Google Calendar API client for data sources and resources.
func (p *googleCalendarProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config googleCalendarProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []option.ClientOption

	// Add credential source
	if !config.Credentials.IsNull() && !config.Credentials.IsUnknown() {
		credentials := config.Credentials.ValueString()
		if credentials != "" {
			opts = append(opts, option.WithCredentialsJSON([]byte(credentials)))
		}
	}

	// Use a custom user-agent string
	terraformVersion := req.TerraformVersion
	if terraformVersion == "" {
		terraformVersion = "unknown"
	}
	userAgent := fmt.Sprintf("(%s %s) Terraform/%s",
		runtime.GOOS, runtime.GOARCH, terraformVersion)
	opts = append(opts, option.WithUserAgent(userAgent))

	// Create the calendar service
	calendarSvc, err := calendar.NewService(ctx, opts...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Google Calendar API client",
			fmt.Sprintf("Failed to create calendar service: %s", err),
		)
		return
	}
	calendarSvc.UserAgent = userAgent

	// Make the calendar service available to resources and data sources
	resp.ResourceData = &Config{
		calendar: calendarSvc,
	}
}

// Resources returns the provider's resources.
func (p *googleCalendarProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewEventResource,
	}
}

// DataSources returns the provider's data sources.
func (p *googleCalendarProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
