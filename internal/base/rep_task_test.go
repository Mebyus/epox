package base

import (
	"testing"
	"time"
)

func TestDailySchedule_Next(t *testing.T) {
	tests := []struct {
		name string

		prev     string
		daydelay uint32

		now  string
		want string
	}{
		{
			name: "1 no delay",

			prev:     "2026-10-05 10:36:00 +05:00",
			daydelay: 0,

			now:  "2026-10-05 10:36:00 +05:00",
			want: "2026-10-06 10:36:00 +05:00",
		},
		{
			name: "2 small delay",

			prev:     "2026-10-05 10:36:00 +05:00",
			daydelay: 0,

			now:  "2026-10-05 11:45:13 +05:00",
			want: "2026-10-06 10:36:00 +05:00",
		},
		{
			name: "3 several days",

			prev:     "2026-10-05 10:36:00 +05:00",
			daydelay: 4,

			now:  "2026-10-05 11:45:13 +05:00",
			want: "2026-10-10 10:36:00 +05:00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const layout = "2006-01-02 15:04:05 Z07:00"

			prev, err := time.Parse(layout, tt.prev)
			if err != nil {
				t.Errorf("parse prev trigger time: %v", err)
				return
			}
			now, err := time.Parse(layout, tt.now)
			if err != nil {
				t.Errorf("parse now instant: %v", err)
				return
			}
			want, err := time.Parse(layout, tt.want)
			if err != nil {
				t.Errorf("parse want instant: %v", err)
				return
			}

			s := NewDailySchedule(prev, tt.daydelay)
			got := s.Next(now)
			if got.Location() != prev.Location() || got.Compare(want) != 0 {
				t.Errorf("Next() = \"%s\", want \"%s\"", got.Format(layout), want.Format(layout))
			}
		})
	}
}
