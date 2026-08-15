package googlecalendar

import (
	"testing"
	"time"

	"google.golang.org/api/calendar/v3"
)

func TestNextOccurrenceBoundary(t *testing.T) {
	// A weekly series anchored on a Friday, observed from a point roughly
	// midway through its run - mirrors the "eric" 1:1 in the sibling repo.
	start := &calendar.EventDateTime{DateTime: "2026-08-07T14:30:00-04:00"}
	recurrence := []string{"RRULE:FREQ=WEEKLY"}

	boundary, err := nextOccurrenceBoundary(recurrence, start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boundary == nil {
		t.Fatal("expected a boundary, got nil")
	}

	// Whatever the next Friday 14:30 ET occurrence at-or-after "now" is, the
	// boundary must land exactly one second before it, and always in UTC.
	if boundary.Location() != time.UTC {
		t.Errorf("boundary not in UTC: %v", boundary.Location())
	}
	if boundary.Second() != 59 {
		t.Errorf("expected a :59 boundary second (one before :00), got %d", boundary.Second())
	}
}

func TestNextOccurrenceBoundary_SeriesEnded(t *testing.T) {
	start := &calendar.EventDateTime{DateTime: "2020-01-03T14:30:00-05:00"}
	recurrence := []string{"RRULE:FREQ=WEEKLY;UNTIL=20200201T000000Z"}

	boundary, err := nextOccurrenceBoundary(recurrence, start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boundary != nil {
		t.Errorf("expected nil boundary for an already-ended series, got %v", boundary)
	}
}

func TestCapRecurrenceUntil(t *testing.T) {
	boundary := time.Date(2026, 8, 18, 3, 59, 59, 0, time.UTC)

	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "plain weekly",
			in:   []string{"RRULE:FREQ=WEEKLY"},
			want: []string{"RRULE:FREQ=WEEKLY;UNTIL=20260818T035959Z"},
		},
		{
			name: "replaces an existing UNTIL",
			in:   []string{"RRULE:FREQ=WEEKLY;UNTIL=20200101T000000Z"},
			want: []string{"RRULE:FREQ=WEEKLY;UNTIL=20260818T035959Z"},
		},
		{
			name: "strips COUNT rather than combining with UNTIL",
			in:   []string{"RRULE:FREQ=WEEKLY;COUNT=10"},
			want: []string{"RRULE:FREQ=WEEKLY;UNTIL=20260818T035959Z"},
		},
		{
			name: "leaves non-RRULE lines alone",
			in:   []string{"RRULE:FREQ=WEEKLY", "EXDATE:20260811T183000Z"},
			want: []string{"RRULE:FREQ=WEEKLY;UNTIL=20260818T035959Z", "EXDATE:20260811T183000Z"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := capRecurrenceUntil(c.in, boundary)
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("line %d: got %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}
