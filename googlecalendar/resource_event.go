package googlecalendar

import (
	"context"
	"fmt"
	"path"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/api/calendar/v3"
)

// Ensure the implementation satisfies the resource.Resource interface.
var _ resource.Resource = &eventResource{}

// eventResource is the resource implementation.
type eventResource struct {
	config *Config
}

// eventResourceModel describes the resource data model.
type eventResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Summary                 types.String `tfsdk:"summary"`
	Location                types.String `tfsdk:"location"`
	Description             types.String `tfsdk:"description"`
	Start                   types.String `tfsdk:"start"`
	End                     types.String `tfsdk:"end"`
	Timezone                types.String `tfsdk:"timezone"`
	GuestsCanInviteOthers   types.Bool   `tfsdk:"guests_can_invite_others"`
	GuestsCanModify         types.Bool   `tfsdk:"guests_can_modify"`
	GuestsCanSeeOtherGuests types.Bool   `tfsdk:"guests_can_see_other_guests"`
	ShowAsAvailable         types.Bool   `tfsdk:"show_as_available"`
	SendNotifications       types.Bool   `tfsdk:"send_notifications"`
	Visibility              types.String `tfsdk:"visibility"`
	Recurrence              types.List   `tfsdk:"recurrence"`
	Conference              types.Map    `tfsdk:"conference"`
	Attendees               types.Set    `tfsdk:"attendee"`
	Attachments             types.Set    `tfsdk:"attachment"`
	HTMLLink                types.String `tfsdk:"html_link"`
}

// attendeeModel describes the attendee nested object.
type attendeeModel struct {
	Email    types.String `tfsdk:"email"`
	Optional types.Bool   `tfsdk:"optional"`
}

// attachmentModel describes the attachment nested object.
type attachmentModel struct {
	FileURL  types.String `tfsdk:"file_url"`
	MimeType types.String `tfsdk:"mime_type"`
	Title    types.String `tfsdk:"title"`
}

// NewEventResource creates a new event resource.
func NewEventResource() resource.Resource {
	return &eventResource{}
}

// Metadata returns the resource type name.
func (r *eventResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event"
}

// Schema defines the schema for the resource.
func (r *eventResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Google Calendar event.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Terraform resource ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"summary": schema.StringAttribute{
				Description: "The summary or title of the event.",
				Required:    true,
			},
			"location": schema.StringAttribute{
				Description: "Geographic location of the event.",
				Optional:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the event.",
				Optional:    true,
			},
			"start": schema.StringAttribute{
				Description: "The start time of the event in RFC3339 format.",
				Required:    true,
			},
			"end": schema.StringAttribute{
				Description: "The end time of the event in RFC3339 format.",
				Required:    true,
			},
			"timezone": schema.StringAttribute{
				Description: "The time zone of the event.",
				Required:    true,
			},
			"guests_can_invite_others": schema.BoolAttribute{
				Description: "Whether attendees can invite others to the event.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"guests_can_modify": schema.BoolAttribute{
				Description: "Whether attendees can modify the event.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"guests_can_see_other_guests": schema.BoolAttribute{
				Description: "Whether attendees can see who else is invited.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"show_as_available": schema.BoolAttribute{
				Description: "Whether to show the time as available (transparent) or busy (opaque).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"send_notifications": schema.BoolAttribute{
				Description: "Whether to send notifications about the event changes.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"visibility": schema.StringAttribute{
				Description: "Visibility of the event.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.OneOf("public", "private", ""),
				},
			},
			"recurrence": schema.ListAttribute{
				Description: "List of RRULE, EXRULE, RDATE and EXDATE lines for a recurring event.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"conference": schema.MapAttribute{
				Description: "Conference data for the event.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"html_link": schema.StringAttribute{
				Description: "An absolute link to the event in the Google Calendar Web UI.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"attendee": schema.SetNestedBlock{
				Description: "The attendees of the event.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"email": schema.StringAttribute{
							Description: "The email address of the attendee.",
							Required:    true,
						},
						"optional": schema.BoolAttribute{
							Description: "Whether this is an optional attendee.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
					},
				},
			},
			"attachment": schema.SetNestedBlock{
				Description: "File attachments for the event.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"file_url": schema.StringAttribute{
							Description: "URL link to the attachment.",
							Required:    true,
						},
						"mime_type": schema.StringAttribute{
							Description: "Internet media type (MIME type) of the attachment.",
							Required:    true,
						},
						"title": schema.StringAttribute{
							Description: "Attachment title.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *eventResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Config, got: %T", req.ProviderData),
		)
		return
	}

	r.config = config
}

// Create creates the resource and sets the initial Terraform state.
func (r *eventResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eventResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the event
	event, diags := r.buildEvent(ctx, &plan, &calendar.Event{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the event via API
	sendNotifications := plan.SendNotifications.ValueBool()
	eventAPI, err := r.config.calendar.Events.
		Insert("primary", event).
		SupportsAttachments(true).
		ConferenceDataVersion(1).
		SendNotifications(sendNotifications).
		MaxAttendees(25).
		Do()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating event",
			fmt.Sprintf("Could not create event: %s", err),
		)
		return
	}

	// Set the ID
	plan.ID = types.StringValue(eventAPI.Id)

	// Read the event to populate computed fields
	r.readEvent(ctx, &plan, eventAPI)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *eventResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state eventResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the event from the API
	event, err := r.config.calendar.Events.
		Get("primary", state.ID.ValueString()).
		Do()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading event",
			fmt.Sprintf("Could not read event %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	// Update the state with the API data
	r.readEvent(ctx, &state, event)

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eventResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan eventResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the current event from the API
	event, err := r.config.calendar.Events.
		Get("primary", plan.ID.ValueString()).
		Do()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading event for update",
			fmt.Sprintf("Could not read event %s: %s", plan.ID.ValueString(), err),
		)
		return
	}

	// Build the updated event
	event, diags := r.buildEvent(ctx, &plan, event)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the event via API
	sendNotifications := plan.SendNotifications.ValueBool()
	eventAPI, err := r.config.calendar.Events.
		Update("primary", plan.ID.ValueString(), event).
		SupportsAttachments(true).
		ConferenceDataVersion(1).
		SendNotifications(sendNotifications).
		MaxAttendees(25).
		Do()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating event",
			fmt.Sprintf("Could not update event %s: %s", plan.ID.ValueString(), err),
		)
		return
	}

	// Update the state with the API data
	r.readEvent(ctx, &plan, eventAPI)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eventResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eventResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the event via API
	sendNotifications := state.SendNotifications.ValueBool()
	err := r.config.calendar.Events.
		Delete("primary", state.ID.ValueString()).
		SendNotifications(sendNotifications).
		Do()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting event",
			fmt.Sprintf("Could not delete event %s: %s", state.ID.ValueString(), err),
		)
		return
	}
}

// buildEvent builds a calendar.Event from the Terraform model.
func (r *eventResource) buildEvent(ctx context.Context, model *eventResourceModel, event *calendar.Event) (*calendar.Event, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Set basic fields
	event.Summary = model.Summary.ValueString()
	event.Location = model.Location.ValueString()
	event.Description = model.Description.ValueString()

	guestsCanInviteOthers := model.GuestsCanInviteOthers.ValueBool()
	event.GuestsCanInviteOthers = &guestsCanInviteOthers
	event.GuestsCanModify = model.GuestsCanModify.ValueBool()
	guestsCanSeeOtherGuests := model.GuestsCanSeeOtherGuests.ValueBool()
	event.GuestsCanSeeOtherGuests = &guestsCanSeeOtherGuests

	showAsAvailable := model.ShowAsAvailable.ValueBool()
	event.Transparency = boolToTransparency(showAsAvailable)
	event.Visibility = model.Visibility.ValueString()

	// Set date/time fields
	event.Start = &calendar.EventDateTime{
		DateTime: model.Start.ValueString(),
		TimeZone: model.Timezone.ValueString(),
	}
	event.End = &calendar.EventDateTime{
		DateTime: model.End.ValueString(),
		TimeZone: model.Timezone.ValueString(),
	}

	// Set recurrence
	if !model.Recurrence.IsNull() {
		var recurrence []string
		diags = append(diags, model.Recurrence.ElementsAs(ctx, &recurrence, false)...)
		event.Recurrence = recurrence
	}

	// Set conference data
	if !model.Conference.IsNull() && !model.Conference.IsUnknown() {
		var conference map[string]string
		diags = append(diags, model.Conference.ElementsAs(ctx, &conference, false)...)
		if googleMeetID, ok := conference["google_meet_id"]; ok && googleMeetID != "" {
			event.ConferenceData = &calendar.ConferenceData{
				ConferenceSolution: &calendar.ConferenceSolution{
					Key: &calendar.ConferenceSolutionKey{
						Type: "hangoutsMeet",
					},
				},
				EntryPoints: []*calendar.EntryPoint{
					{
						EntryPointType: "video",
						Label:          fmt.Sprintf("meet.google.com/%s", googleMeetID),
						Uri:            fmt.Sprintf("https://meet.google.com/%s", googleMeetID),
					},
				},
			}
		}
	}

	// Set attendees
	if !model.Attendees.IsNull() && !model.Attendees.IsUnknown() {
		var attendees []attendeeModel
		diags = append(diags, model.Attendees.ElementsAs(ctx, &attendees, false)...)

		attendeesExisting := event.Attendees
		apiAttendees := make([]*calendar.EventAttendee, len(attendees))

		for i, att := range attendees {
			apiAttendees[i] = &calendar.EventAttendee{
				Email: att.Email.ValueString(),
			}
			// If attendee is already on the event, preserve their existing attributes
			for _, ea := range attendeesExisting {
				if ea.Email == apiAttendees[i].Email {
					apiAttendees[i] = ea
					break
				}
			}
			// Set the optional field (this is managed by the provider)
			apiAttendees[i].Optional = att.Optional.ValueBool()
		}

		event.Attendees = apiAttendees
	}

	// Set attachments
	if !model.Attachments.IsNull() && !model.Attachments.IsUnknown() {
		var attachments []attachmentModel
		diags = append(diags, model.Attachments.ElementsAs(ctx, &attachments, false)...)

		apiAttachments := make([]*calendar.EventAttachment, len(attachments))
		for i, att := range attachments {
			apiAttachments[i] = &calendar.EventAttachment{
				FileUrl:  att.FileURL.ValueString(),
				MimeType: att.MimeType.ValueString(),
				Title:    att.Title.ValueString(),
			}
		}

		event.Attachments = apiAttachments
	}

	return event, diags
}

// readEvent updates the Terraform model from a calendar.Event.
func (r *eventResource) readEvent(ctx context.Context, model *eventResourceModel, event *calendar.Event) {
	model.Summary = types.StringValue(event.Summary)

	// Optional string attributes - treat empty strings as null
	if event.Location != "" {
		model.Location = types.StringValue(event.Location)
	} else {
		model.Location = types.StringNull()
	}

	if event.Description != "" {
		model.Description = types.StringValue(event.Description)
	} else {
		model.Description = types.StringNull()
	}

	if event.Start != nil {
		model.Start = types.StringValue(event.Start.DateTime)
		model.Timezone = types.StringValue(event.Start.TimeZone)
	}
	if event.End != nil {
		model.End = types.StringValue(event.End.DateTime)
	}

	if event.GuestsCanInviteOthers != nil {
		model.GuestsCanInviteOthers = types.BoolValue(*event.GuestsCanInviteOthers)
	}
	model.GuestsCanModify = types.BoolValue(event.GuestsCanModify)
	if event.GuestsCanSeeOtherGuests != nil {
		model.GuestsCanSeeOtherGuests = types.BoolValue(*event.GuestsCanSeeOtherGuests)
	}

	model.ShowAsAvailable = types.BoolValue(transparencyToBool(event.Transparency))
	model.Visibility = types.StringValue(event.Visibility)

	// Set recurrence
	if len(event.Recurrence) > 0 {
		recurrenceList := make([]attr.Value, len(event.Recurrence))
		for i, r := range event.Recurrence {
			recurrenceList[i] = types.StringValue(r)
		}
		model.Recurrence, _ = types.ListValue(types.StringType, recurrenceList)
	} else {
		model.Recurrence = types.ListNull(types.StringType)
	}

	// Set conference data
	if event.ConferenceData != nil && len(event.ConferenceData.EntryPoints) > 0 {
		conferenceMap := map[string]attr.Value{
			"google_meet_id": types.StringValue(path.Base(event.ConferenceData.EntryPoints[0].Uri)),
		}
		model.Conference, _ = types.MapValue(types.StringType, conferenceMap)
	} else {
		model.Conference = types.MapNull(types.StringType)
	}

	// Set attendees
	if len(event.Attendees) > 0 {
		attendeeObjectType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"email":    types.StringType,
				"optional": types.BoolType,
			},
		}
		attendeeList := make([]attr.Value, len(event.Attendees))
		for i, att := range event.Attendees {
			attendeeList[i], _ = types.ObjectValue(
				attendeeObjectType.AttrTypes,
				map[string]attr.Value{
					"email":    types.StringValue(att.Email),
					"optional": types.BoolValue(att.Optional),
				},
			)
		}
		model.Attendees, _ = types.SetValue(attendeeObjectType, attendeeList)
	} else {
		model.Attendees = types.SetNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"email":    types.StringType,
				"optional": types.BoolType,
			},
		})
	}

	// Set attachments
	if len(event.Attachments) > 0 {
		attachmentObjectType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"file_url":  types.StringType,
				"mime_type": types.StringType,
				"title":     types.StringType,
			},
		}
		attachmentList := make([]attr.Value, len(event.Attachments))
		for i, att := range event.Attachments {
			// Handle optional title field - treat empty as null
			var title attr.Value
			if att.Title != "" {
				title = types.StringValue(att.Title)
			} else {
				title = types.StringNull()
			}

			attachmentList[i], _ = types.ObjectValue(
				attachmentObjectType.AttrTypes,
				map[string]attr.Value{
					"file_url":  types.StringValue(att.FileUrl),
					"mime_type": types.StringValue(att.MimeType),
					"title":     title,
				},
			)
		}
		model.Attachments, _ = types.SetValue(attachmentObjectType, attachmentList)
	} else {
		model.Attachments = types.SetNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"file_url":  types.StringType,
				"mime_type": types.StringType,
				"title":     types.StringType,
			},
		})
	}

	// Set computed fields
	model.HTMLLink = types.StringValue(event.HtmlLink)
}

// boolToTransparency converts a boolean representing "show as available" to the
// corresponding transparency string.
func boolToTransparency(showAsAvailable bool) string {
	if !showAsAvailable {
		return "opaque"
	}
	return "transparent"
}

// transparencyToBool converts a transparency string into a boolean representing
// "show as available".
func transparencyToBool(s string) bool {
	return s == "transparent"
}
