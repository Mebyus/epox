package hits

import (
	"net/http"
)

type Router interface {
	Dispatch(*Context) ([]Handler, error)
}

type RouteTree struct {
	// root for patterns with GET method
	get *pnode

	post *pnode
}

var _ Router = &RouteTree{}

func (r *RouteTree) Dispatch(c *Context) ([]Handler, error) {
	path, err := ParsePath(c.Req.URL.Path)
	if err != nil {
		return nil, err
	}

	var p *pattern
	var h []Handler
	switch c.Req.Method {
	case http.MethodGet:
		if r.get != nil {
			p, h = r.get.match(path.Parts, 0, path.Slash)
		}
	case http.MethodPost:
		if r.post != nil {
			p, h = r.post.match(path.Parts, 0, path.Slash)
		}
	}

	// TODO: save path params
	_ = p

	return h, nil
}

type Simple struct {
	get    []route
	post   []route
	patch  []route
	put    []route
	delete []route
	query  []route

	IgnoreTrailingSlash bool
}

var _ Router = &Simple{}

func (s *Simple) Dispatch(c *Context) ([]Handler, error) {
	path, err := ParsePath(c.Req.URL.Path)
	if err != nil {
		return nil, err
	}
	if s.IgnoreTrailingSlash {
		path.Slash = false
	}

	var r *route
	switch c.Req.Method {
	case http.MethodGet:
		r = match(s.get, path)
	case http.MethodPost:
		r = match(s.post, path)
	case http.MethodPatch:
		r = match(s.patch, path)
	case http.MethodPut:
		r = match(s.put, path)
	case http.MethodDelete:
		r = match(s.delete, path)
	case "QUERY":
		r = match(s.query, path)
	}
	if r == nil {
		return nil, nil
	}

	// save named path params
	if r.params != 0 {
		if c.Params == nil {
			c.Params = make(map[string]string, r.params)
		}

		for i, seg := range r.segments {
			if seg.typ == segName {
				name := seg.val
				c.Params[name] = path.Parts[i]
			}
		}
	}

	return r.chain, nil
}

func match(routes []route, path *Path) *route {
	for i := range len(routes) {
		r := &routes[i]
		if r.pattern.matchPath(path) {
			return r
		}
	}
	return nil
}

// Attach handler to request path pattern.
//
// Pattern specifies request path pattern that can be later
// matched against an incoming request.
//
// Each pattern has form:
//
//	METHOD PATH
//
//	METHOD - one of http methods (GET, POST, PUT, ...)
//	PATH - segmented (by /) path that starts with /
//
// PATH can contain wildcards in form:
//
//	{name} - matches one segment and stores it under given name
//	*      - matches one segment and discards it
//	**     - matches any number of segments and discards them
func (r *RouteTree) Attach(pattern string, handler Handler, other ...Handler) {

}

// path node for pattern tree
type pnode struct {
	// always not empty for leaf node
	chain []Handler

	// child nodes
	//
	// array contains list of child nodes for each segment type
	nodes [4][]pnode

	seg segment

	// not empty only for leaf node
	pattern *pattern
}

//	/api/test
//	/api/test/company
//	/api/{name}/company
//	/api/*/device
//
// parts - all not empty parts of request path separated by /
// pos   - index of current part being matched
// slash - true for paths ending with /
func (n *pnode) match(parts []string, pos int, slash bool) (*pattern, []Handler) {
	if n.pattern != nil {
		if n.pattern.matchParts(parts, pos, slash) {
			return n.pattern, n.chain
		}
	}

	if pos >= len(parts) {
		return nil, nil
	}
	if n.seg.typ == segLit && n.seg.val != parts[pos] {
		return nil, nil
	}

	for _, nodes := range n.nodes {
		for _, node := range nodes {
			p, c := node.match(parts, pos+1, slash)
			if p != nil {
				return p, c
			}
		}
	}

	return nil, nil
}

type route struct {
	pattern

	chain []Handler
}

// returns true if parts match segments starting from
// specified position (index) and trailing slash
func (p *pattern) matchParts(parts []string, pos int, slash bool) bool {
	if p.slash != slash {
		return false
	}
	if pos > len(p.segments) || len(parts) != len(p.segments) {
		return false
	}

	for i := pos; i < len(parts); i += 1 {

	}
	return true
}

func matchPath(pats []*pattern, path Path) bool {
	for _, p := range pats {
		p.matchParts(path.Parts, 0, p.slash)
	}
	return false
}
