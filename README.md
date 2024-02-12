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

When this provider changes or destoys a recurring event, all instances past and
future are impacted. In many cases, this is not what you want.

If you only want to change or remove future events, the easiest thing to do is
delete the next occurance on the web UI and select "this and following events".
From here, remove the event from state and create a new one.

What this does to an event is add an `UNTIL` rule to `recurrence`. For example
if you have an event configured with `RRULE:FREQ=WEEKLY`, it will be updated to
something like `RRULE:FREQ=WEEKLY;UNTIL=20240220T045959Z`. This will show up in
execution plans.

If you attempt to only change the event with "this and following events", an
`UNTIL` rule is added just as if you deleted it. Google then creates a
completely new event that would not be managed with Terraform. This will also
happen if `guests_can_modify` is true and someone else performs a similar
action.

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
