// Package dagvis visualizes a DAG graph into
// a structured, layered planer map.
package dagvis

// Graph is a directed graph
type Graph struct {
	Nodes map[string][]string
}

// Reverse the graph
func (g *Graph) Reverse() *Graph {
	ret := make(map[string][]string)

	for n := range g.Nodes {
		ret[n] = nil // touch every node first
	}

	for n, lst := range g.Nodes {
		for _, m := range lst {
			ret[m] = append(ret[m], n)
		}
	}

	return &Graph{Nodes: ret}
}

// Rename renames the name of each node in the graph
func (g *Graph) Rename(f func(string) (string, error)) (*Graph, error) {
	if f == nil {
		panic("rename function is nil")
	}

	ret := new(Graph)
	ret.Nodes = make(map[string][]string)

	for k, vs := range g.Nodes {
		newK, e := f(k)
		if e != nil {
			return nil, e
		}

		newVs := make([]string, 0, len(vs))
		for _, v := range vs {
			newV, e := f(v)
			if e != nil {
				return nil, e
			}

			newVs = append(newVs, newV)
		}

		ret.Nodes[newK] = newVs
	}

	return ret, nil
}
