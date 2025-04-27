package segmenttree

import (
	"strings"

	"github.com/codecrafters-io/http-server-starter-go/app/types"
)

type SegmentNode struct {
	children          map[string]*SegmentNode
	parameterChildren map[string]*SegmentNode
	paramName         string
	handlers          map[types.Method]types.Handler
	isEndOfPath       bool
}

type SegmentTree struct {
	root *SegmentNode
}

func NewSegmentTree() *SegmentTree {
	return &SegmentTree{
		root: &SegmentNode{
			children:          make(map[string]*SegmentNode),
			handlers:          make(map[types.Method]types.Handler),
			parameterChildren: make(map[string]*SegmentNode),
		},
	}
}

func newNode() *SegmentNode {
	return &SegmentNode{
		children:          make(map[string]*SegmentNode),
		handlers:          make(map[types.Method]types.Handler),
		parameterChildren: make(map[string]*SegmentNode),
	}
}

func (t *SegmentTree) Insert(method types.Method, path string, handler types.Handler) {
	segments := strings.Split(path, "/")
	node := t.root
	for _, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			name := strings.TrimPrefix(seg, ":")
			child, ok := node.parameterChildren[name]
			if !ok {
				child = newNode()
				child.paramName = name
				node.parameterChildren[name] = child
			}
			node = child
		} else {
			child, ok := node.children[seg]
			if !ok {
				child = newNode()
				node.children[seg] = child
			}
			node = child
		}
	}
	node.isEndOfPath = true
	node.handlers[method] = handler
}

func (t *SegmentTree) Search(method types.Method, path string) (types.Handler, map[string]string, bool) {
	segments := strings.Split(path, "/")
	params := make(map[string]string)
	h, ok := t.searchNode(t.root, segments, method, params)
	if !ok {
		return nil, nil, false
	}
	return h, params, true
}

func (t *SegmentTree) searchNode(node *SegmentNode, segments []string, method types.Method, params map[string]string) (types.Handler, bool) {
	if len(segments) == 0 {
		if !node.isEndOfPath {
			return nil, false
		}
		h, ok := node.handlers[method]
		return h, ok
	}
	seg := segments[0]
	rest := segments[1:]

	if child, exists := node.children[seg]; exists {
		if h, ok := t.searchNode(child, rest, method, params); ok {
			return h, true
		}
	}
	if seg != "" {
		for name, child := range node.parameterChildren {
			params[name] = seg
			if h, ok := t.searchNode(child, rest, method, params); ok {
				return h, true
			}
			delete(params, name)
		}
	}
	return nil, false
}
