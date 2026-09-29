package hits

import (
	"io"
	"strings"
	"testing"
)

func TestLimitedBodyReader_Read(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		limit int64
	}{
		{
			name:  "1 empty string (zero limit)",
			body:  "",
			limit: 0,
		},
		{
			name:  "2 empty string (non-zero limit)",
			body:  "",
			limit: 6,
		},
		{
			name:  "3 non-empty string (zero limit)",
			body:  "hello",
			limit: 0,
		},
		{
			name:  "4 non-empty string (limit less than body length)",
			body:  "hello",
			limit: 3,
		},
		{
			name:  "5 non-empty string (limit equal to body length)",
			body:  "hello",
			limit: 5,
		},
		{
			name:  "6 non-empty string (limit greater than body length)",
			body:  "hello",
			limit: 10,
		},
		{
			name:  "7 big string (limit less than body length)",
			body:  strings.Repeat("hello", 1<<14),
			limit: 5*(1<<14) - 15,
		},
		{
			name:  "8 big string (limit equal to body length)",
			body:  strings.Repeat("hello", 1<<14),
			limit: 5 * (1 << 14),
		},
		{
			name:  "9 big string (limit greater than body length)",
			body:  strings.Repeat("hello", 1<<14),
			limit: 5*(1<<14) + 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLimitReader(io.NopCloser(strings.NewReader(tt.body)), tt.limit)
			wantLimitErr := tt.limit < int64(len(tt.body))

			buf := strings.Builder{}
			_, err := io.Copy(&buf, l)
			if err != nil && !wantLimitErr {
				t.Errorf("Copy() error = <%v>, want error <false>", err)
				return
			}
			if err != nil && wantLimitErr {
				if err != ErrBodyLimitReached {
					t.Errorf("Copy() error = <%v>, want error <%v>", err, ErrBodyLimitReached)
					return
				}
			}

			var want string
			if wantLimitErr {
				want = tt.body[:tt.limit]
			} else {
				want = tt.body
			}

			got := buf.String()
			if got != want {
				t.Errorf("Copy() got = \"%s\", want \"%s\"", got, want)
				return
			}
		})
	}
}
