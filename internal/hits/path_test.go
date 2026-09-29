package hits

import "testing"

func TestParsePath(t *testing.T) {
	tests := []struct {
		name    string
		p       string
		want    Path
		wantErr bool
	}{
		{
			name: "1 empty path",
			p:    "",
			want: Path{},
		},
		{
			name: "2 relative path",
			p:    "abc",
			want: Path{Parts: []string{"abc"}},
		},
		{
			name: "3 root path",
			p:    "/",
			want: Path{Abs: true, Slash: true},
		},
		{
			name: "4 root path extra slash",
			p:    "//",
			want: Path{Abs: true, Slash: true},
		},
		{
			name: "5 one part",
			p:    "/a",
			want: Path{Parts: []string{"a"}, Abs: true},
		},
		{
			name: "6 one part trailing slash",
			p:    "/a/",
			want: Path{Parts: []string{"a"}, Abs: true, Slash: true},
		},
		{
			name: "7 one part many slashes",
			p:    "///a////",
			want: Path{Parts: []string{"a"}, Abs: true, Slash: true},
		},
		{
			name: "8 two parts",
			p:    "/a/abc",
			want: Path{Parts: []string{"a", "abc"}, Abs: true},
		},
		{
			name: "9 three parts",
			p:    "/a/abc/xc12/",
			want: Path{Parts: []string{"a", "abc", "xc12"}, Abs: true, Slash: true},
		},
		{
			name: "10 empty parts",
			p:    "//a///abc/xz////34a//",
			want: Path{Parts: []string{"a", "abc", "xz", "34a"}, Abs: true, Slash: true},
		},
		{
			name: "11 relative with empty parts",
			p:    "a///abc/xz////34a//",
			want: Path{Parts: []string{"a", "abc", "xz", "34a"}, Slash: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := ParsePath(tt.p)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ParsePath() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ParsePath() succeeded unexpectedly")
			}
			if !got.Equal(&tt.want) {
				t.Errorf("ParsePath() = \"%s\", want \"%s\"", got, &tt.want)
			}
		})
	}
}
