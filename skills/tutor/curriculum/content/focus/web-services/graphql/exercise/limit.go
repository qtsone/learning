package gql

// Limits is the budget a single query may spend. Zero or negative disables a
// check, which is a decision you should have to type on purpose.
type Limits struct {
	MaxDepth      int
	MaxComplexity int
}

// Depth is how deeply a query nests. A field in the operation's own selection
// set is at depth 1, and every nested selection set adds one:
//
//	{ posts { title } }                  -> 2
//	{ post(id: "p1") { author { name } } } -> 3
//
// Depth is cheap to compute and cheap to reason about, and it is the only
// thing standing between a cyclic schema (Post.author.posts.author…) and a
// query that never ends.
func Depth(op *Operation) int {
	// TODO: return the deepest nesting in op.Selections.
	return 0
}

// Complexity estimates what a query will cost to serve, before serving it.
//
// The model, exactly as the tests expect it:
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
	// TODO: walk op.Selections from s.Root(op.Type) accumulating the cost
	// above. Skip fields the schema does not define (Validate has already
	// rejected them by the time this runs in the handler).
	return 0
}

// Check rejects a query that exceeds either budget. Depth first: it is the
// cheaper computation and the clearer message.
//
// The error is a request error — the query never runs — so callers answer it
// with 400 and no data.
func (l Limits) Check(s *Schema, op *Operation) error {
	// TODO: report "query is too deep: depth D exceeds the limit of N" and
	// "query is too complex: complexity C exceeds the limit of N", and nil
	// when the query fits (or when a limit is not positive).
	return nil
}
