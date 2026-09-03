package gql

import (
	"context"
	"errors"
	"fmt"
)

// Args are a field's arguments, already parsed into Go values (int, string,
// bool). A real server type-checks these against the schema's argument
// definitions; this one keeps the two accessors it needs.
type Args map[string]any

func (a Args) Int(name string) (int, bool) {
	n, ok := a[name].(int)
	return n, ok
}

func (a Args) String(name string) (string, bool) {
	s, ok := a[name].(string)
	return s, ok
}

// Resolver fetches one field, for one parent object. That signature is the
// whole GraphQL execution model: there is no "query for this request", only a
// tree of field resolutions, each ignorant of the others. Everything expensive
// about GraphQL follows from it.
type Resolver func(ctx context.Context, parent any, args Args) (any, error)

// FieldDef is one field of one object type — the hand-written equivalent of a
// line of SDL plus the resolver bound to it.
type FieldDef struct {
	Name string

	// Object is the name of the object type this field returns, or "" for a
	// scalar. List says whether it returns many of them.
	Object string
	List   bool

	// Weight is what one resolution of this field costs the complexity
	// analyser. The default is 1; raise it for a field you know is expensive
	// however well you batch it.
	Weight int

	// CountArg names the argument that bounds a list field (here: "first"),
	// and PageSize is what the analyser assumes when the client omits it.
	// Without PageSize, `posts { … }` would look free to the analyser and
	// return the whole table at runtime.
	CountArg string
	PageSize int

	Resolve Resolver
}

// ObjectDef is one object type: a name and its fields.
type ObjectDef struct {
	Name   string
	Fields map[string]*FieldDef
}

// Schema is the executable form of schema.graphql.
type Schema struct {
	Types    map[string]*ObjectDef
	Query    *ObjectDef
	Mutation *ObjectDef
}

// Root returns the object type an operation starts from.
func (s *Schema) Root(t OperationType) *ObjectDef {
	if t == OperationMutation {
		return s.Mutation
	}
	return s.Query
}

// List converts a typed slice into the []any a list resolver must return. The
// executor cannot walk a []Post without reflection, and reflection in the hot
// path of every field is a price this package declines to pay.
func List[T any](vs []T) []any {
	out := make([]any, len(vs))
	for i, v := range vs {
		out[i] = v
	}
	return out
}

// applyFirst honours the argument the complexity analyser charged for. A
// resolver that ignores `first` turns the analyser's estimate into a lie.
func applyFirst[T any](vs []T, args Args) []T {
	if n, ok := args.Int("first"); ok && n >= 0 && n < len(vs) {
		return vs[:n]
	}
	return vs
}

// resolverSet holds the four resolvers that differ between the naive schema
// and the batched one. Everything else about the two schemas is identical,
// which is the point: N+1 is not a modelling mistake, it is a fetching one.
type resolverSet struct {
	postAuthor    Resolver
	postComments  Resolver
	commentAuthor Resolver
	authorPosts   Resolver
}

// NewSchema builds the schema whose nested resolvers batch through the
// per-request dataloaders.
func NewSchema(store *Store) *Schema { return buildSchema(store, batchedResolvers()) }

// NaiveSchema builds the same schema with nested resolvers that fetch one
// parent at a time. It is correct, it is what everybody writes first, and it
// is the N+1 problem.
func NaiveSchema(store *Store) *Schema { return buildSchema(store, naiveResolvers(store)) }

func buildSchema(store *Store, r resolverSet) *Schema {
	post := &ObjectDef{Name: "Post", Fields: map[string]*FieldDef{
		"id":    {Name: "id", Resolve: scalar(func(p Post) any { return p.ID })},
		"title": {Name: "title", Resolve: scalar(func(p Post) any { return p.Title })},
		"author": {
			Name: "author", Object: "Author",
			Resolve: r.postAuthor,
		},
		"comments": {
			Name: "comments", Object: "Comment", List: true,
			CountArg: "first", PageSize: 20,
			Resolve: r.postComments,
		},
	}}

	comment := &ObjectDef{Name: "Comment", Fields: map[string]*FieldDef{
		"id":   {Name: "id", Resolve: scalar(func(c Comment) any { return c.ID })},
		"body": {Name: "body", Resolve: scalar(func(c Comment) any { return c.Body })},
		"author": {
			Name: "author", Object: "Author",
			Resolve: r.commentAuthor,
		},
	}}

	author := &ObjectDef{Name: "Author", Fields: map[string]*FieldDef{
		"id":   {Name: "id", Resolve: scalar(func(a Author) any { return a.ID })},
		"name": {Name: "name", Resolve: scalar(func(a Author) any { return a.Name })},
		"posts": {
			Name: "posts", Object: "Post", List: true,
			CountArg: "first", PageSize: 20,
			Resolve: r.authorPosts,
		},
	}}

	query := &ObjectDef{Name: "Query", Fields: map[string]*FieldDef{
		"posts": {
			Name: "posts", Object: "Post", List: true,
			CountArg: "first", PageSize: 20,
			Resolve: func(ctx context.Context, _ any, args Args) (any, error) {
				ps, err := store.Posts(ctx)
				if err != nil {
					return nil, err
				}
				return List(applyFirst(ps, args)), nil
			},
		},
		"post": {
			Name: "post", Object: "Post",
			Resolve: func(ctx context.Context, _ any, args Args) (any, error) {
				id, _ := args.String("id")
				p, err := store.PostByID(ctx, id)
				if errors.Is(err, ErrNotFound) {
					// A nullable field answering "no such thing" with null and
					// no error is not a failure: the client asked, and the
					// answer is nothing.
					return nil, nil
				}
				if err != nil {
					return nil, err
				}
				return p, nil
			},
		},
	}}

	mutation := &ObjectDef{Name: "Mutation", Fields: map[string]*FieldDef{
		"createPost": {
			Name: "createPost", Object: "Post",
			Resolve: func(ctx context.Context, _ any, args Args) (any, error) {
				authorID, ok := args.String("authorId")
				if !ok {
					return nil, fmt.Errorf("createPost: authorId is required")
				}
				title, ok := args.String("title")
				if !ok {
					return nil, fmt.Errorf("createPost: title is required")
				}
				return store.CreatePost(ctx, authorID, title)
			},
		},
	}}

	return &Schema{
		Types: map[string]*ObjectDef{
			"Query": query, "Mutation": mutation,
			"Post": post, "Comment": comment, "Author": author,
		},
		Query:    query,
		Mutation: mutation,
	}
}

// scalar adapts a plain field accessor to the Resolver signature. Most fields
// of most schemas are exactly this: read a struct field, return it.
func scalar[T any](get func(T) any) Resolver {
	return func(_ context.Context, parent any, _ Args) (any, error) {
		p, ok := parent.(T)
		if !ok {
			return nil, fmt.Errorf("resolver got parent of type %T", parent)
		}
		return get(p), nil
	}
}
