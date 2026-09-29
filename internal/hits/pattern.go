package hits

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// pattern is a parsed, validated form of route pattern "METHOD PATH".
// Examples of valid patterns:
//
//		GET   /info
//		GET   /alarm/*
//		POST  /user/{id}/message
//
//	 POST   user/login
//	 POST   user/logout
//		GET    user/info/
//
// Method and path are optional in pattern.
//
// zero value of pattern is valid and represents empty pattern
type pattern struct {
	// only empty for path = "/" or ""
	// otherwise always contains at least one element
	segments []segment

	// always a valid http method
	//
	// may be empty if pattern represents only path, without method
	method string

	// total number of named path param segments
	params uint32

	// total number of wildcard path segments
	wilds uint32

	// pattern starts with "/"
	root bool

	// pattern ends with "/"
	slash bool

	// true when pattern is not empty i.e. has non-empty string representation
	// false when pattern represents empty string
	ne bool
}

func (p *pattern) String() string {
	if p == nil || !p.ne {
		return ""
	}

	var g strings.Builder
	n := 0                 // total number of bytes in resulting string
	n += 1 + len(p.method) // method + space

	n += len(p.segments) // add necessary space for "/" before each segment
	if len(p.segments) == 0 {
		if p.root {
			n += 1
		}
	} else {
		if !p.root {
			n -= 1
		}
		if p.slash {
			n += 1
		}
	}

	for _, seg := range p.segments {
		switch seg.typ {
		case segLit:
			n += len(seg.val)
		case segName:
			n += 2 + len(seg.val)
		case segWild:
			n += 1
		case segAny:
			n += 2
		}
	}

	if p.method != "" {
		_, _ = g.WriteString(p.method)
		_ = g.WriteByte(' ')
	}

	if p.root {
		_ = g.WriteByte('/')
	}
	if len(p.segments) == 0 {
		return g.String()
	}

	writeSegment(&g, p.segments[0])
	for _, seg := range p.segments[1:] {
		_ = g.WriteByte('/')
		writeSegment(&g, seg)
	}
	if p.slash {
		_ = g.WriteByte('/')
	}

	return g.String()
}

func writeSegment(g *strings.Builder, seg segment) {
	switch seg.typ {
	case segLit:
		_, _ = g.WriteString(seg.val)
	case segName:
		_ = g.WriteByte('{')
		_, _ = g.WriteString(seg.val)
		_ = g.WriteByte('}')
	case segWild:
		_ = g.WriteByte('*')
	case segAny:
		_ = g.WriteByte('*')
		_ = g.WriteByte('*')
	}
}

func (p *pattern) matchPath(path *Path) bool {
	if p.root != path.Abs || p.slash != path.Slash {
		return false
	}
	if len(p.segments) != len(path.Parts) {
		return false
	}

	for i := range len(p.segments) {
		s := p.segments[i]
		part := path.Parts[i]

		if s.typ == segLit && s.val != part {
			return false
		}
	}
	return true
}

// indicates segment type
type segType uint8

const (
	// literal
	segLit segType = iota

	// named wildcard (path parameter)
	segName

	// wildcard *
	segWild

	// wildcard **
	segAny
)

type segment struct {
	val string // literal, wildcard name or empty for * or ** wildcards
	typ segType
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t'
}

// cuts a string into two strings
//
// first string that comes before space
// second string that comes after space
//
// space is one or more space characters
//
// if there are no spaces or space only comes
// at the end of the string, then second string
// is returned empty
func cutSpace(s string) (string, string) {
	i := 0 // index into pattern string
	for i < len(s) {
		if isSpace(s[i]) {
			break
		}
		i += 1
	}
	a := s[:i]

	for i < len(s) {
		if !isSpace(s[i]) {
			break
		}
		i += 1
	}
	return a, s[i:]
}

// parse pattern from its string form.
//
// Empty string produces empty pattern.
// When parsing these pattern forms are checked (in the exact order):
//
//   - empty pattern
//   - METHOD PATH
//   - PATH
//
// Thus non-empty string can never produce a pattern with
// non-empty method and empty path.
func parsePattern(s string) (*pattern, error) {
	if s == "" {
		return &pattern{}, nil
	}
	method, path := cutSpace(s)
	if path == "" {
		path = method
		method = ""
	}

	p := pattern{
		method: method,
		ne:     true,
	}
	err := parsePatternPath(&p, path)
	if err != nil {
		return nil, fmt.Errorf("parse path: %v", err)
	}

	return &p, nil
}

func parsePatternPath(pat *pattern, s string) error {
	if s == "" {
		return nil
	}

	// remove leading and trailing slashes
	{
		var slash bool
		if s[len(s)-1] == '/' {
			slash = true

			i := len(s) - 2
			for i >= 0 && s[i] == '/' {
				i -= 1
			}
			if i < 0 {
				pat.root = true
				return nil
			}

			s = s[:i+1]
		}

		i := 0
		for i < len(s) && s[i] == '/' {
			i += 1
		}
		s = s[i:]
		pat.slash = slash
		pat.root = i > 0
	}

	parts := strings.Split(s, "/")
	segments := make([]segment, 0, len(parts))
	var params uint32
	var wilds uint32
	for _, part := range parts {
		if part == "" {
			continue
		}

		if part == "*" {
			segments = append(segments, segment{typ: segWild})
			wilds += 1
			continue
		}
		if part == "**" {
			segments = append(segments, segment{typ: segAny})
			continue
		}

		if part[0] == '{' {
			if len(part) < len("{a}") {
				return errors.New("invalid named segment")
			}

			if part[len(part)-1] != '}' {
				return errors.New("invalid named segment")
			}

			name := part[1 : len(part)-1] // not empty due to length check above
			if !isValidSegmentName(name) {
				return fmt.Errorf("invalid name \"%s\" in segment", name)
			}

			segments = append(segments, segment{
				val: name,
				typ: segName,
			})
			params += 1
			continue
		}

		segments = append(segments, segment{
			val: part,
			typ: segLit,
		})
	}
	if len(segments) != 0 {
		pat.segments = segments
		pat.params = params
		pat.wilds = wilds
	}

	return nil
}

// join two patterns into a new one
//
// in joined pattern path segments are concatenated
// and method is the first non-empty method between
// the two
func join(a, b *pattern) *pattern {
	if a == nil {
		if b == nil {
			return &pattern{}
		}
		a = &pattern{}
	}
	if b == nil {
		b = &pattern{}
	}
	if !a.ne && !b.ne {
		return &pattern{}
	}

	var method string
	if a.method != "" {
		method = a.method
	} else {
		method = b.method
	}

	var root bool
	if a.ne {
		root = a.root
	} else {
		root = b.root
	}

	var slash bool
	if b.ne {
		slash = b.slash
	} else {
		slash = a.slash
	}

	return &pattern{
		segments: append(a.segments, b.segments...),
		params:   a.params + b.params,
		wilds:    a.wilds + b.wilds,
		root:     root,
		slash:    slash,
		method:   method,
		ne:       true,
	}
}

func isValidMethod(s string) bool {
	switch s {
	case http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodPatch, http.MethodHead,
		http.MethodTrace, http.MethodOptions, http.MethodConnect:
		return true
	default:
		return false
	}
}

func isValidSegmentName(name string) bool {
	for i := range len(name) {
		c := name[i]
		if !isValidChar(c) {
			return false
		}
	}
	return true
}

func isValidChar(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9') || c == '-' || c == '_' || c == '.'
}
