package rush

import (
	"net/http"
	"slices"
	"strings"
)

type node struct {
	segment       string
	children      map[string]*node
	paramChild    *node
	wildcardChild *node
	handlers      map[string]http.Handler
	allowHeader   string
}

func newNode(segment string) *node {
	return &node{
		segment:  segment,
		children: make(map[string]*node),
		handlers: make(map[string]http.Handler),
	}
}

func (n *node) nextOrCreate(segment string) *node {
	if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
		name := segment[1 : len(segment)-1]
		if name == "" {
			panic("empty param name not allowed")
		}
		if n.paramChild == nil {
			n.paramChild = newNode(name)
		}
		return n.paramChild
	}

	if segment == "*" {
		if n.wildcardChild == nil {
			n.wildcardChild = newNode("*")
		}
		return n.wildcardChild
	}

	next, ok := n.children[segment]
	if ok {
		return next
	}
	next = newNode(segment)
	n.children[segment] = next
	return next
}

func (n *node) allow() string {
	if n.allowHeader == "" {
		methods := make([]string, 0, len(n.handlers)+1)
		for method := range n.handlers {
			methods = append(methods, method)
		}
		methods = append(methods, http.MethodOptions)
		slices.Sort(methods)
		n.allowHeader = strings.Join(methods, ", ")
	}
	return n.allowHeader
}

type trie struct {
	root *node
}

func splitPath(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool { return r == '/' })
}

func (t *trie) insert(pattern string, handler http.Handler, methods ...string) {
	segments := splitPath(pattern)
	cur := t.root
	for i, seg := range segments {
		if seg == "*" && i != len(segments)-1 {
			panic("wildcard '*' can only be at the end of the route")
		}
		cur = cur.nextOrCreate(seg)
	}
	for _, method := range methods {
		cur.handlers[method] = handler
	}
}

type routeMatch struct {
	node   *node
	params map[string]string
}

func (t *trie) lookup(path string) (routeMatch, bool) {
	segments := splitPath(path)
	params := make(map[string]string)
	return t.root.match(0, segments, params)
}

func (n *node) match(i int, segments []string, params map[string]string) (routeMatch, bool) {
	if i == len(segments) {
		return routeMatch{node: n, params: params}, true
	}

	seg := segments[i]
	i++

	if child, ok := n.children[seg]; ok {
		if m, ok := child.match(i, segments, params); ok {
			return m, ok
		}
	}

	if n.paramChild != nil {
		params[n.paramChild.segment] = seg
		if m, ok := n.paramChild.match(i, segments, params); ok {
			return m, ok
		}
		delete(params, n.paramChild.segment)
	}

	if n.wildcardChild != nil {
		return routeMatch{node: n.wildcardChild, params: params}, true
	}

	return routeMatch{}, false
}
