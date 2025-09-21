package rush

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

type node struct {
	children      map[string]*node
	handlers      map[string]http.Handler
	paramChild    *node
	wildcardChild *node
	segment       string
	allowHeader   string
}

func newNode(segment string) *node {
	return &node{
		segment:  segment,
		children: make(map[string]*node, 4),
		handlers: make(map[string]http.Handler, 2),
	}
}

func (n *node) nextOrCreate(segment string) *node {
	if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
		name := segment[1 : len(segment)-1]
		if name == "" {
			panic("rush: empty parameter name '{}' is not allowed")
		}
		if n.paramChild == nil {
			n.paramChild = newNode(name)
		}
		if n.paramChild.segment != name {
			panic(fmt.Sprintf(
				"rush: parameter name conflict - cannot use both '{%s}' and '{%s}' at the same path level",
				n.paramChild.segment,
				name,
			))
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
		if _, ok := n.handlers[http.MethodOptions]; !ok {
			methods = append(methods, http.MethodOptions)
		}

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
			panic("rush: wildcard '*' can only be at the end of the route")
		}
		cur = cur.nextOrCreate(seg)
	}
	for _, method := range methods {
		cur.handlers[method] = handler
	}
}

func (t *trie) lookup(path string, r *http.Request) *node {
	return t.root.match(1, path, r)
}

func (n *node) match(i int, path string, r *http.Request) *node {
	if i >= len(path) {
		if len(n.handlers) > 0 {
			return n
		}
		return nil
	}

	end := i
	for end < len(path) && path[end] != '/' {
		end++
	}
	segment := path[i:end]
	next := end + 1

	if child, ok := n.children[segment]; ok {
		if m := child.match(next, path, r); m != nil {
			return m
		}
	}

	if n.paramChild != nil {
		r.SetPathValue(n.paramChild.segment, segment)
		if m := n.paramChild.match(next, path, r); m != nil {
			return m
		}
		r.SetPathValue(n.paramChild.segment, "")
	}

	if n.wildcardChild != nil {
		return n.wildcardChild
	}

	return nil
}
