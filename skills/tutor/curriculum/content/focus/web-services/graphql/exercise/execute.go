package gql

import (
	"context"
	"fmt"
)

// QueryError is one entry of the response's errors array. Path is where the
// failure happened in the result tree — ["posts", 4, "author"] — which is the
// only thing that makes a partial response debuggable.
type QueryError struct {
	Message string `json:"message"`
	Path    []any  `json:"path,omitempty"`
}

// Response is the GraphQL response envelope. Data is omitted entirely when the
// request failed before execution began (a parse error, a rejected query);
// once execution starts, data is present even if every field in it is null.
type Response struct {
	Data   map[string]any `json:"data,omitempty"`
	Errors []QueryError   `json:"errors,omitempty"`
}

// Deferred is a value that is not available yet. A resolver returns one when
// it has queued a key with a dataloader instead of fetching immediately; the
// executor forces it after the loaders have been dispatched.
type Deferred func() (any, error)

// Validate checks an operation against the schema before anything runs.
// Everything it reports is a request error: the client sent a query this
// schema cannot answer, so nothing executes and the response carries no data.
func Validate(s *Schema, op *Operation) error {
	root := s.Root(op.Type)
	if root == nil {
		return fmt.Errorf("this schema has no %s type", op.Type)
	}
	return validateSelections(s, root, op.Selections)
}

func validateSelections(s *Schema, obj *ObjectDef, sels []*Field) error {
	for _, f := range sels {
		def, ok := obj.Fields[f.Name]
		if !ok {
			return fmt.Errorf("cannot query field %q on type %q", f.Name, obj.Name)
		}
		if def.Object == "" {
			if len(f.Selections) > 0 {
				return fmt.Errorf("field %q on type %q is a scalar and has no subfields", f.Name, obj.Name)
			}
			continue
		}
		if len(f.Selections) == 0 {
			return fmt.Errorf("field %q on type %q must have a selection of subfields", f.Name, obj.Name)
		}
		if err := validateSelections(s, s.Types[def.Object], f.Selections); err != nil {
			return err
		}
	}
	return nil
}

// task is one field waiting to be resolved for one parent value.
type task struct {
	def    *FieldDef
	field  *Field
	parent any
	path   []any
	assign func(any)
}

// Execute runs a validated operation one *level* at a time: every field at the
// current depth is resolved, then the dataloaders are dispatched, then the
// results are unpacked into the next level's tasks.
//
// That level boundary is the whole trick. A resolver called once per parent
// cannot batch anything on its own; what it can do is queue a key and hand
// back a Deferred, and the executor's "all resolvers at this depth have run"
// moment is when one batch call can serve all of them.
//
// Production servers reach the same point differently: they run sibling
// resolvers concurrently and give the loader a short time window to collect
// keys. That works, and it makes every test about batching a race against a
// timer. This executor is single-goroutine and dispatches at an explicit
// point, so counting store calls is deterministic — which is why the tests can
// assert on them at all.
func Execute(ctx context.Context, s *Schema, op *Operation) *Response {
	data := map[string]any{}
	resp := &Response{Data: data}
	root := s.Root(op.Type)

	// Top-level mutation fields must run in series (a client that asks for two
	// mutations expects the first to finish first). Resolving one level with a
	// sequential loop gives that for free.
	level := fieldTasks(root, op.Selections, nil, nil, data)

	type resolved struct {
		t   task
		val any
		err error
	}

	for len(level) > 0 {
		if err := ctx.Err(); err != nil {
			resp.Errors = append(resp.Errors, QueryError{Message: err.Error()})
			return resp
		}

		batch := make([]resolved, 0, len(level))
		for _, t := range level {
			v, err := t.def.Resolve(ctx, t.parent, t.field.Args)
			batch = append(batch, resolved{t: t, val: v, err: err})
		}

		Dispatch(ctx)

		var next []task
		for _, r := range batch {
			val, err := r.val, r.err
			if d, ok := val.(Deferred); ok && err == nil {
				val, err = d()
			}
			if err != nil {
				// The field becomes null and the query keeps going: partial
				// data is the normal GraphQL outcome, not a special case.
				resp.Errors = append(resp.Errors, QueryError{Message: err.Error(), Path: r.t.path})
				r.t.assign(nil)
				continue
			}
			children, qerr := expand(s, r.t, val)
			if qerr != nil {
				resp.Errors = append(resp.Errors, *qerr)
				r.t.assign(nil)
				continue
			}
			next = append(next, children...)
		}
		level = next
	}
	return resp
}

// expand writes a resolved value into the response and returns the tasks its
// sub-selection needs.
func expand(s *Schema, t task, val any) ([]task, *QueryError) {
	if val == nil {
		t.assign(nil)
		return nil, nil
	}
	if t.def.Object == "" {
		t.assign(val)
		return nil, nil
	}
	child := s.Types[t.def.Object]
	if !t.def.List {
		out := map[string]any{}
		t.assign(out)
		return fieldTasks(child, t.field.Selections, val, t.path, out), nil
	}
	items, ok := val.([]any)
	if !ok {
		return nil, &QueryError{
			Message: fmt.Sprintf("resolver for %q returned %T, want []any (wrap it in List)", t.def.Name, val),
			Path:    t.path,
		}
	}
	out := make([]any, len(items))
	var next []task
	for i, item := range items {
		m := map[string]any{}
		out[i] = m
		next = append(next, fieldTasks(child, t.field.Selections, item, extend(t.path, i), m)...)
	}
	t.assign(out)
	return next, nil
}

func fieldTasks(obj *ObjectDef, sels []*Field, parent any, path []any, into map[string]any) []task {
	out := make([]task, 0, len(sels))
	for _, f := range sels {
		key := f.Name
		out = append(out, task{
			def:    obj.Fields[key],
			field:  f,
			parent: parent,
			path:   extend(path, key),
			assign: func(v any) { into[key] = v },
		})
	}
	return out
}

// extend always allocates: sibling tasks must not share a path array, or the
// error paths of a wide selection set overwrite each other.
func extend(path []any, step any) []any {
	out := make([]any, len(path)+1)
	copy(out, path)
	out[len(path)] = step
	return out
}
