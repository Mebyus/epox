package hits

import "testing"

func Test_parsePattern(t *testing.T) {
	tests := []struct {
		name       string
		p          string
		want       string
		wantParams uint32
		wantWilds  uint32
		wantErr    bool
	}{
		{
			name: "1 empty pattern",
			p:    "",
			want: "",
		},
		{
			name: "2 only path",
			p:    "KK",
			want: "KK",
		},
		{
			name: "3 path with segments",
			p:    "/abc/zz1/",
			want: "/abc/zz1/",
		},
		{
			name: "4 relative path one segment",
			p:    "POST abc",
		},
		{
			name: "5 root path",
			p:    "POST /",
		},
		{
			name: "6 root path one segment",
			p:    "POST /abc",
		},
		{
			name: "7 root path one segment trailing slash",
			p:    "POST /abc/",
		},
		{
			name: "8 extra slashes",
			p:    "POST ///abc///",
			want: "POST /abc/",
		},
		{
			name:       "9 root path with param",
			p:          "POST /abc/{id}",
			wantParams: 1,
		},
		{
			name:       "10 root path with wildcards",
			p:          "POST /abc/*/{id}/**",
			wantParams: 1,
		},
		{
			name:       "11 relative path with params",
			p:          "POST /abc/{id}/{name}/",
			wantParams: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pat, gotErr := parsePattern(tt.p)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("parsePattern() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("parsePattern() succeeded unexpectedly")
			}

			got := pat.String()
			want := tt.want
			if want == "" {
				want = tt.p
			}
			if got != want {
				t.Errorf("pattern = \"%s\", want \"%s\"", got, want)
			}
			if pat.params != tt.wantParams {
				t.Errorf("params = %d, want %d", pat.params, tt.wantParams)
			}
		})
	}
}

func Test_cutSpace(t *testing.T) {
	tests := []struct {
		name string
		s    string
		a    string
		b    string
	}{
		{
			name: "1 empty string",
			s:    "",
			a:    "",
			b:    "",
		},
		{
			name: "2 single byte string",
			s:    "a",
			a:    "a",
			b:    "",
		},
		{
			name: "3 multibyte string",
			s:    "abc",
			a:    "abc",
			b:    "",
		},
		{
			name: "4 space start",
			s:    " abc",
			a:    "",
			b:    "abc",
		},
		{
			name: "5 space end",
			s:    "abc ",
			a:    "abc",
			b:    "",
		},
		{
			name: "6 space start end",
			s:    "  abc ",
			a:    "",
			b:    "abc ",
		},
		{
			name: "7 two words",
			s:    "abc xx",
			a:    "abc",
			b:    "xx",
		},
		{
			name: "8 tabs",
			s:    "abc \t \txx",
			a:    "abc",
			b:    "xx",
		},
		{
			name: "9 three words",
			s:    "abc xx qq",
			a:    "abc",
			b:    "xx qq",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := cutSpace(tt.s)
			if got != tt.a || got2 != tt.b {
				t.Errorf("cutSpace() = (\"%s\", \"%s\"), want (%s, %s)", got, got2, tt.a, tt.b)
			}
		})
	}
}

func Test_join(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		a    *pattern
		b    *pattern
		want *pattern
	}{
		{
			name: "1 nils",
			a:    nil,
			b:    nil,
			want: &pattern{},
		},
		{
			name: "2 nil+root",
			a:    nil,
			b:    &pattern{ne: true, root: true},
			want: &pattern{ne: true, root: true},
		},
		{
			name: "3 root+abc",
			a:    &pattern{ne: true, root: true, slash: true},
			b:    &pattern{ne: true, segments: []segment{{val: "abc"}, {val: "zzz"}}},
			want: &pattern{ne: true, root: true, segments: []segment{{val: "abc"}, {val: "zzz"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := join(tt.a, tt.b)

			gots := got.String()
			wants := tt.want.String()

			if gots != wants {
				t.Errorf("join() = %s, want %s", gots, wants)
			}
		})
	}
}
