package gql

import "fmt"

// Limits is the budget a single query may spend. Zero or negative disables a
// check, which is a decision you should have to type on purpose.
type Limits struct {
	MaxDepth      int
	MaxComplexity int
}

// Depth is how deeply a query nests. A field in the operation's own selection
// set is at depth 1, and every nested selection set adds one:
//
//	{ posts { title } }                    -> 2
//	{ post(id: "p1") { author { name } } } -> 3
//
// Depth is cheap to compute and cheap to reason about, and it is the only
// thing standing between a cyclic schema (Post.author.posts.author…) and a
// query that never ends.
func Depth(op *Operation) int { return selectionDepth(op.Selections) }

func selectionDepth(sels []*Field) int {
	deepest := 0
	for _, f := range sels {
		d := 1
		if len(f.Selections) > 0 {
			d += selectionDepth(f.Selections)
		}
		if d > deepest {
			deepest = d
		}
	}
	return deepest
}

// Complexity estimates what a query will cost to serve, before serving it.
//
// The model:
//
//   - a field is resolved once per parent object, so it costs
//     Weight × (number of parents), where Weight defaults to 1;
//   - a list field multiplies the number of parents for everything *inside*
//     it, by the value of its CountArg argument if the client supplied one,
//     and by its PageSize if not;
//   - the operation's own fields start with a single parent.
//
// So `{ posts(first: 3) { title } }` costs 1 for posts plus 3 for the three
// titles: 4. And `{ posts(first: 100) { comments(first: 100) { author { name
// } } } }` costs 1 + 100 + 10000 + 10000 = 20101, from five lines of query.
//
// That last number is the argument for this whole file. Depth alone would rate
// that query a harmless 4.
func Complexity(s *Schema, op *Operation) int {
	return selectionCost(s, s.Root(op.Type), op.Selections, 1)
}

func selectionCost(s *Schema, obj *ObjectDef, sels []*Field, parents int) int {
	if obj == nil {
		return 0
	}
	total := 0
	for _, f := range sels {
		def, ok := obj.Fields[f.Name]
		if !ok {
			continue // Validate has already rejected this query
		}
		weight := def.Weight
		if weight <= 0 {
			weight = 1
		}
		total += weight * parents
		if len(f.Selections) == 0 {
			continue
		}
		children := parents
		if def.List {
			children = parents * listSize(def, f.Args)
		}
		total += selectionCost(s, s.Types[def.Object], f.Selections, children)
	}
	return total
}

// listSize is how many items the analyser assumes a list field will return:
// what the client asked for, or the field's page size when it asked for
// nothing. A list field with neither is charged as one item, which is a schema
// bug rather than a client one — every list field wants a bound.
func listSize(def *FieldDef, args Args) int {
	if def.CountArg != "" {
		if n, ok := args.Int(def.CountArg); ok && n >= 0 {
			return n
		}
	}
	if def.PageSize > 0 {
		return def.PageSize
	}
	return 1
}

// Check rejects a query that exceeds either budget. Depth first: it is the
// cheaper computation and the clearer message.
//
// The error is a request error — the query never runs — so callers answer it
// with 400 and no data.
func (l Limits) Check(s *Schema, op *Operation) error {
	if l.MaxDepth > 0 {
		if d := Depth(op); d > l.MaxDepth {
			return fmt.Errorf("query is too deep: depth %d exceeds the limit of %d", d, l.MaxDepth)
		}
	}
	if l.MaxComplexity > 0 {
		if c := Complexity(s, op); c > l.MaxComplexity {
			return fmt.Errorf("query is too complex: complexity %d exceeds the limit of %d", c, l.MaxComplexity)
		}
	}
	return nil
}
