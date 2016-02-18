// Package treemux is a generic treemux ripped from httptreemux.
package treemux

import (
	"fmt"
)

type TreeMux struct {
	root *node
}

func (t *TreeMux) Dump() string {
	return t.root.dumpTree("", "")
}

func (t *TreeMux) Set(path string, v interface{}) {
	if path[0] != '/' {
		panic(fmt.Sprintf("Path %s must start with slash", path))
	}

	node := t.root.addPath(path[1:], nil)
	node.setValue(v)
}

func (t *TreeMux) Get(path string) (interface{}, map[string]string) {
	n, params := t.root.search(path[1:])
	if n == nil {
		return nil, nil
	}

	var paramMap map[string]string
	if len(params) != 0 {
		if len(params) != len(n.leafWildcardNames) {
			// Need better behavior here. Should this be a panic?
			panic(fmt.Sprintf("treemux parameter list length mismatch: %v, %v",
				params, n.leafWildcardNames))
		}

		paramMap = make(map[string]string)
		numParams := len(params)
		for index := 0; index < numParams; index++ {
			paramMap[n.leafWildcardNames[numParams-index-1]] = params[index]
		}
	}

	return n.leafValue, paramMap
}

func New() *TreeMux {
	root := &node{path: "/"}
	return &TreeMux{
		root: root,
	}
}
