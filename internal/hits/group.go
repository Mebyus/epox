package hits

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
)

// Group combines routes and child groups into a tree which can
// be later converted into router.
//
// Zero value is a root group ready for use .
type Group struct {
	routes []route

	// child groups
	groups []*Group

	// common handlers for all routes and child groups
	chain []Handler

	// common path prefix for all routes and child groups
	prefix string

	// prefix parsed as path pattern
	//
	// equals nil if prefix is empty
	pp *pattern
}

func NewRoot() *Group {
	return &Group{
		prefix: "/",
		pp: &pattern{
			root:  true,
			slash: true,
			ne:    true,
		},
	}
}

func NewRouter(root *Group) *Simple {
	const debug = false

	if !root.pp.root {
		panic("non-root group used as router root group")
	}
	routes := root.gather(nil, nil, nil)
	s := &Simple{}

	if debug {
		for _, r := range routes {
			fmt.Printf("%s (%d handlers)\n", &r.pattern, len(r.chain))
		}
	}

	for _, r := range routes {
		if !r.root {
			panic(fmt.Sprintf("gathered route \"%s\" with relative path pattern", &r.pattern))
		}

		switch r.method {
		case "":
			panic("empty method")
		case http.MethodGet:
			s.get = append(s.get, r)
		case http.MethodPost:
			s.post = append(s.post, r)
		case http.MethodPatch:
			s.patch = append(s.patch, r)
		case http.MethodPut:
			s.put = append(s.put, r)
		case http.MethodDelete:
			s.delete = append(s.delete, r)
		default:
			panic(fmt.Sprintf("unexpected method %s", r.method))
		}
	}

	sortRoutes(s.get)
	sortRoutes(s.post)
	sortRoutes(s.patch)
	sortRoutes(s.put)
	sortRoutes(s.delete)

	return s
}

func sortRoutes(routes []route) {
	if len(routes) <= 1 {
		return
	}

	slices.SortFunc(routes, func(a, b route) int {
		if a.wilds < b.wilds {
			return -1
		}
		if a.wilds > b.wilds {
			return 1
		}
		if a.params < b.params {
			return -1
		}
		if a.params > b.params {
			return 1
		}
		return cmp.Compare(len(a.segments), len(b.segments))
	})
}

// Add new route to group.
func (g *Group) Add(method, path string, handler Handler, chain ...Handler) {
	if method == "" {
		panic("empty method")
	}
	r := route{
		pattern: pattern{
			method: method,
			ne:     true,
		},
		chain: append([]Handler{handler}, chain...),
	}
	err := parsePatternPath(&r.pattern, path)
	if err != nil {
		panic(err)
	}
	g.routes = append(g.routes, r)
}

// Group creates new child group from this one.
//
// Prefix specifies common path prefix for all routes
// and child groups.
func (g *Group) Group(prefix string, chain ...Handler) *Group {
	p := &pattern{}
	err := parsePatternPath(p, prefix)
	if err != nil {
		panic(err)
	}
	child := &Group{
		chain:  chain,
		prefix: prefix,
		pp:     p,
	}
	g.groups = append(g.groups, child)
	return child
}

// Recursive method which walks group subtree.
//
// Append all routes from the group and its subtree to the given
// slice and return the resulting slice
func (g *Group) gather(routes []route, prefix *pattern, chain []Handler) []route {
	newPrefix := join(prefix, g.pp)
	newChain := append(chain, g.chain...)
	for _, r := range g.routes {
		routes = append(routes, route{
			pattern: *join(newPrefix, &r.pattern),
			chain:   append(newChain, r.chain...),
		})
	}
	for _, group := range g.groups {
		routes = group.gather(routes, newPrefix, newChain)
	}

	return routes
}
