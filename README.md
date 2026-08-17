# Terraform Google Calendar Provider

This fork is optimized for managing recurring meetings, like one on ones.

It's not intended to be an all-encompassing Google Calendar provider.

## Installation

This is not deployed to the terraform registry. For now you'd have to clone this
repo, build the provider locally (`go build`), and reference the provider
locally.

You can use a `.terraformrc` file like this:

```hcl
plugin_cache_dir   = "$HOME/.terraform.d/plugin-cache"
disable_checkpoint = true

provider_installation {
  dev_overrides {
    "googlecalendar" = "/path/to/terraform-provider-googlecalendar"
  }
}
```

See [docs](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers)
for more info on .terraformrc.

With the above in a `.terraformrc`, you can reference the provider like this:

```hcl
terraform {
  required_providers {
    googlecalendar = {
      source = "googlecalendar"
    }
  }
}
```

## Usage

```hcl
resource "googlecalendar_event" "test" {

  summary     = "Test Terraform Event"
  description = "Testing fork of sethvargo/terraform-provider-googlecalendar"

  # RFC3339 date format - https://datatracker.ietf.org/doc/html/rfc3339
  #
  # UTC offset is optional since we also have the `timezone` argument - though
  # necessary if you intend to use the `timeadd` function
  start = "2023-12-27T20:00:00"
  end   = "2023-12-27T21:00:00"

  # IANA time zone database format - https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
  #
  # Google supports having different start and end timezones, but this provider does not.
  timezone = "America/New_York"

  # RFC5545 format for recurrence
  #
  # https://datatracker.ietf.org/doc/html/rfc5545#section-3.8.5.3
  recurrence = [
    "RRULE:FREQ=WEEKLY",
  ]

  attendee {
    email = "me@domain.com"
  }

  attendee {
    email = "you@domain.com"
  }

  attachment {
    title     = "you : me"
    file_url  = "https://docs.google.com/document/d/.../edit"
    mime_type = "application/vnd.google-apps.document"
  }

  conference = {

    # Only Google Meet is supported so far.
    #
    # This provider doesn't attempt to create new instances of meetings.
    # Go to meet.google.com and "create a meeting for later".
    google_meet_id = "aaa-bbbb-ccc"
  }
}
```

### Changing Events

When this provider updates a recurring event in place, all instances past and
future are impacted. Some changes - like moving a series to a different
weekday - aren't sensible as an in-place update at all, since they change which
occurrences the series was ever meant to cover. For those, destroy and
recreate the resource, e.g.:

```
terraform apply -replace='googlecalendar_event.main["someone"]'
```

By default (`deletion_policy = "DELETE"`, the default), destroying a resource
deletes the entire series, past and future. Set `deletion_policy = "TRUNCATE"`
to instead cap the series with an `UNTIL` just before its next occurrence -
equivalent to the calendar UI's "this and following events" delete - so past
instances stay on the calendar. Terraform still drops the resource from state
either way; only the underlying calendar event differs.

```hcl
resource "googlecalendar_event" "someone" {
  # ...
  deletion_policy = "TRUNCATE"
}
```

This implements the approach described in
[this guide](https://developers.google.com/calendar/api/guides/recurringevents#modifying_all_following_instances)
for "this and following events" edits.

### Reconciling External Forks

The "this and following events" split isn't unique to `deletion_policy =
"TRUNCATE"` - anyone editing the event directly in the Calendar UI can trigger
the same split on a series Terraform manages. That caps the tracked event with
an `UNTIL` and forks the remainder onto a new id, all outside Terraform. The
next `plan` or `apply` then sees the cap as recurrence drift, but updating the
tracked id in place would just remove the cap from an event Google has already
superseded - it wouldn't touch the live series.

Set `auto_reconcile = true` to have `Read` detect this - an `UNTIL` appearing
on the tracked event's recurrence that wasn't there last time - and search
forward from the cap for the live continuation (a same-summary event carrying
a different recurring event id), repointing state at it automatically:

```hcl
resource "googlecalendar_event" "someone" {
  # ...
  auto_reconcile = true
}
```

Either way, a `Warning` diagnostic is emitted so the outcome shows up in
`plan`/`apply` output - the id being repointed and where to review it, or why
no continuation was found (the series may have simply ended at the cap).

It's off by default: a wrong match would silently repoint state at the wrong
event, and unlike `deletion_policy`, there's no explicit action here to signal
intent - it's watching for something that can happen at any time, outside of
Terraform.

### Importing Existing Events

You can import existing Google Calendar events into Terraform state using the
event's ID:

```bash
terraform import googlecalendar_event.my_meeting <event-id>
```

## Google Authentication

Anticipated use is with `gcloud` using your own Google identity with Application
Default Credentials.

Your ADC will require an additional scope. This command would log you in and set
defaults + calendar access:

```sh
gcloud auth login

gcloud auth application-default login \
  --billing-project "$BILLING_PROJECT" \
  --scopes \
openid,\
https://www.googleapis.com/auth/userinfo.email,\
https://www.googleapis.com/auth/cloud-platform,\
https://www.googleapis.com/auth/sqlservice.login,\
https://www.googleapis.com/auth/calendar
```

`$BILLING_PROJECT` must be set to a GCP project where the
`calendar-json.googleapis.com` service is enabled.
