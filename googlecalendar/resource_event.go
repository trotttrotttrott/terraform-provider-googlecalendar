package googlecalendar

import (
	"fmt"
	"log"
	"path"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"google.golang.org/api/calendar/v3"
)

var (
	eventValidMethods     = []string{"email", "popup", "sms"}
	eventValidVisbilities = []string{"public", "private"}
)

func resourceEvent() *schema.Resource {
	return &schema.Resource{
		Create: resourceEventCreate,
		Read:   resourceEventRead,
		Update: resourceEventUpdate,
		Delete: resourceEventDelete,

		Schema: map[string]*schema.Schema{
			"summary": {
				Type:     schema.TypeString,
				Required: true,
			},

			"location": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"start": {
				Type:     schema.TypeString,
				Required: true,
			},

			"end": {
				Type:     schema.TypeString,
				Required: true,
			},

			"timezone": {
				Type:     schema.TypeString,
				Required: true,
			},

			"guests_can_invite_others": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"guests_can_modify": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"guests_can_see_other_guests": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"show_as_available": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"send_notifications": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"visibility": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.StringInSlice(eventValidVisbilities, false),
			},

			"recurrence": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"conference": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"google_meet_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"attendee": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"email": {
							Type:     schema.TypeString,
							Required: true,
						},

						"optional": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},

			"attachment": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"file_url": {
							Type:     schema.TypeString,
							Required: true,
						},

						"mime_type": {
							Type:     schema.TypeString,
							Required: true,
						},

						"title": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			//
			// Computed values
			//
			"event_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"html_link": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// resourceEventCreate creates a new event via the API.
func resourceEventCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	event, err := resourceEventBuild(d, meta)
	if err != nil {
		return fmt.Errorf("failed to build event: %w", err)
	}

	eventAPI, err := config.calendar.Events.
		Insert("primary", event).
		SupportsAttachments(true).
		ConferenceDataVersion(1).
		SendNotifications(d.Get("send_notifications").(bool)).
		MaxAttendees(25).
		Do()
	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	d.SetId(eventAPI.Id)

	return resourceEventRead(d, meta)
}

// resourceEventRead reads information about the event from the API.
func resourceEventRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	event, err := config.calendar.Events.
		Get("primary", d.Id()).
		Do()
	if err != nil {
		return fmt.Errorf("failed to read event: %w", err)
	}

	d.Set("summary", event.Summary)
	d.Set("location", event.Location)
	d.Set("description", event.Description)
	d.Set("start", event.Start)
	d.Set("end", event.End)
	d.Set("timezone", event.Start.TimeZone)

	if event.GuestsCanInviteOthers != nil {
		d.Set("guests_can_invite_others", *event.GuestsCanInviteOthers)
	}
	d.Set("guests_can_modify", event.GuestsCanModify)
	if event.GuestsCanSeeOtherGuests != nil {
		d.Set("guests_can_see_other_guests", *event.GuestsCanSeeOtherGuests)
	}

	d.Set("show_as_available", transparencyToBool(event.Transparency))
	d.Set("visibility", event.Visibility)
	d.Set("recurrence", event.Recurrence)

	if event.ConferenceData != nil && len(event.ConferenceData.EntryPoints) > 0 {
		d.Set("conference", map[string]interface{}{
			"google_meet_id": path.Base(event.ConferenceData.EntryPoints[0].Uri),
		})
	}

	if len(event.Attendees) > 0 {
		d.Set("attendee", flattenEventAttendees(event.Attendees))
	}

	if len(event.Attachments) > 0 {
		d.Set("attachment", flattenEventAttachments(event.Attachments))
	}

	d.Set("event_id", event.Id)
	d.Set("html_link", event.HtmlLink)

	return nil
}

// resourceEventUpdate updates an event via the API.
func resourceEventUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	event, err := resourceEventBuild(d, meta)
	if err != nil {
		return fmt.Errorf("failed to build event: %w", err)
	}

	eventAPI, err := config.calendar.Events.
		Update("primary", d.Id(), event).
		SupportsAttachments(true).
		ConferenceDataVersion(1).
		SendNotifications(d.Get("send_notifications").(bool)).
		MaxAttendees(25).
		Do()
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	d.SetId(eventAPI.Id)

	return resourceEventRead(d, meta)
}

// resourceEventDelete deletes an event via the API.
func resourceEventDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	id := d.Id()
	sendNotifications := d.Get("send_notifications").(bool)

	err := config.calendar.Events.
		Delete("primary", id).
		SendNotifications(sendNotifications).
		Do()
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	d.SetId("")

	return nil
}

// resourceBuildEvent is a shared helper function which builds an "event" struct
// from the schema. This is used by create and update.
func resourceEventBuild(d *schema.ResourceData, meta interface{}) (*calendar.Event, error) {
	summary := d.Get("summary").(string)
	location := d.Get("location").(string)
	description := d.Get("description").(string)

	start := d.Get("start").(string)
	end := d.Get("end").(string)
	timezone := d.Get("timezone").(string)

	guestsCanInviteOthers := d.Get("guests_can_invite_others").(bool)
	guestsCanModify := d.Get("guests_can_modify").(bool)
	guestsCanSeeOtherGuests := d.Get("guests_can_see_other_guests").(bool)
	showAsAvailable := d.Get("show_as_available").(bool)
	visibility := d.Get("visibility").(string)
	recurrence := listToStringSlice(d.Get("recurrence").([]interface{}))

	var event calendar.Event
	event.Summary = summary
	event.Location = location
	event.Description = description
	event.GuestsCanInviteOthers = &guestsCanInviteOthers
	event.GuestsCanModify = guestsCanModify
	event.GuestsCanSeeOtherGuests = &guestsCanSeeOtherGuests
	event.Transparency = boolToTransparency(showAsAvailable)
	event.Visibility = visibility
	event.Recurrence = recurrence
	event.Start = &calendar.EventDateTime{
		DateTime: start,
		TimeZone: timezone,
	}
	event.End = &calendar.EventDateTime{
		DateTime: end,
		TimeZone: timezone,
	}

	conference := d.Get("conference").(map[string]interface{})
	if len(conference) > 0 {
		googleMeetID := conference["google_meet_id"].(string)
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

	// Parse attendees
	attendeesRaw := d.Get("attendee").(*schema.Set)
	if attendeesRaw.Len() > 0 {
		attendees := make([]*calendar.EventAttendee, attendeesRaw.Len())

		for i, v := range attendeesRaw.List() {
			m := v.(map[string]interface{})

			attendees[i] = &calendar.EventAttendee{
				Email:    m["email"].(string),
				Optional: m["optional"].(bool),
			}
		}

		event.Attendees = attendees
	}

	// Parse attachments
	attachmentsRaw := d.Get("attachment").(*schema.Set)
	if attachmentsRaw.Len() > 0 {

		attachments := make([]*calendar.EventAttachment, attachmentsRaw.Len())

		for i, v := range attachmentsRaw.List() {
			m := v.(map[string]interface{})

			attachments[i] = &calendar.EventAttachment{
				FileUrl:  m["file_url"].(string),
				MimeType: m["mime_type"].(string),
				Title:    m["title"].(string),
			}
		}

		event.Attachments = attachments
	}

	return &event, nil
}

// flattenEventAttendees flattens the list of event attendees into a map for
// storing in the schema.
func flattenEventAttendees(list []*calendar.EventAttendee) []map[string]interface{} {
	result := make([]map[string]interface{}, len(list))
	for i, v := range list {
		result[i] = map[string]interface{}{
			"email":    v.Email,
			"optional": v.Optional,
		}
	}
	return result
}

func flattenEventAttachments(list []*calendar.EventAttachment) []map[string]interface{} {
	result := make([]map[string]interface{}, len(list))
	for i, v := range list {
		result[i] = map[string]interface{}{
			"file_url":  v.FileUrl,
			"mime_type": v.MimeType,
			"title":     v.Title,
		}
	}
	return result
}

// boolToTransparency converts a boolean representing "show as available" to the
// corresponding transpency string.
func boolToTransparency(showAsAvailable bool) string {
	if !showAsAvailable {
		return "opaque"
	}
	return "transparent"
}

// transparencyToBool converts a transparency string into a boolean representing
// "show as available".
func transparencyToBool(s string) bool {
	switch s {
	case "opaque":
		return false
	case "transparent":
		return true
	default:
		log.Printf("[WARN] unknown transparency %q", s)
		return false
	}
}
