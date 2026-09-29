package hits

import (
	"slices"
	"strings"
)

// Path describes request path in parsed, refined form.
//
// Zero value of this type corresponds to minimal valid relative path "".
type Path struct {
	// Contains list of all non-empty path parts.
	//
	// Each part is a string between consecutive "/" symbols.
	Parts []string

	// True for non-trivial paths which end with at least one "/".
	//
	// As a special case equals true for absolute root path "/".
	Slash bool

	// True for paths which start with at least one "/". Such paths are called
	// absolute. The opposite are relative paths.
	Abs bool
}

func (p *Path) String() string {
	parts := p.Parts

	if len(parts) == 0 {
		if p.Abs {
			return "/"
		}
		return ""
	}

	var g strings.Builder
	n := 0              // total number of bytes in resulting string
	n += len(parts) - 1 // add necessary space for "/" between each part
	for _, part := range parts {
		n += len(part)
	}
	if p.Abs {
		n += 1
	}
	if p.Slash {
		n += 1
	}
	g.Grow(n)

	if p.Abs {
		_ = g.WriteByte('/')
	}
	_, _ = g.WriteString(parts[0])
	for _, part := range parts[1:] {
		_ = g.WriteByte('/')
		_, _ = g.WriteString(part)
	}
	if p.Slash {
		_ = g.WriteByte('/')
	}
	return g.String()
}

func (p *Path) Equal(a *Path) bool {
	if p.Abs != a.Abs || p.Slash != a.Slash {
		return false
	}
	return slices.Equal(p.Parts, a.Parts)
}

// ParsePath transforms a path string into parsed form.
//
// Request path must always start with "/".
func ParsePath(s string) (*Path, error) {
	if s == "" {
		return &Path{}, nil
	}
	if s == "/" {
		return &Path{Abs: true, Slash: true}, nil
	}

	var p Path
	if s[0] == '/' {
		p.Abs = true

		// remove starting slashes
		i := 1
		for i < len(s) && s[i] == '/' {
			i += 1
		}
		s = s[i:]
		if s == "" {
			p.Slash = true
			return &p, nil
		}
	}
	if s[len(s)-1] == '/' {
		p.Slash = true

		// remove trailing slashes
		i := len(s) - 2
		for i >= 0 && s[i] == '/' {
			i -= 1
		}
		s = s[:i+1]
	}

	// extract non-empty parts from remaining string
	var parts []string
	{
		i := 0 // scan index
		j := 0 // prev part index
		for i < len(s) {
			if s[i] == '/' {
				if i > j {
					parts = append(parts, s[j:i])
				}
				j = i + 1
			}
			i += 1
		}
		if i > j { // i == len(p)
			parts = append(parts, s[j:i])
		}

		if len(parts) == 0 {
			return nil, nil
		}
	}
	p.Parts = parts

	return &p, nil
}
