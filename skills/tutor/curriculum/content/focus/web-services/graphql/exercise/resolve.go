package gql

import "context"

// ---------------------------------------------------------------- naive
//
// These four resolvers are correct. Every field returns the right value, the
// tests for shape and errors pass, and a code review would wave them through.
// They are also the N+1 problem: each one fetches for exactly one parent, and
// the executor calls it once per parent.

func naiveResolvers(store *Store) resolverSet {
	return resolverSet{
		postAuthor: func(ctx context.Context, parent any, _ Args) (any, error) {
			return store.AuthorByID(ctx, parent.(Post).AuthorID)
		},
		commentAuthor: func(ctx context.Context, parent any, _ Args) (any, error) {
			return store.AuthorByID(ctx, parent.(Comment).AuthorID)
		},
		postComments: func(ctx context.Context, parent any, args Args) (any, error) {
			cs, err := store.CommentsByPost(ctx, parent.(Post).ID)
			if err != nil {
				return nil, err
			}
			return List(applyFirst(cs, args)), nil
		},
		authorPosts: func(ctx context.Context, parent any, args Args) (any, error) {
			ps, err := store.PostsByAuthor(ctx, parent.(Author).ID)
			if err != nil {
				return nil, err
			}
			return List(applyFirst(ps, args)), nil
		},
	}
}

// ---------------------------------------------------------------- batched
//
// The same four fields, fetched through the request's dataloaders. Nothing
// about the schema changes; only where the data comes from.

func batchedResolvers() resolverSet {
	return resolverSet{
		postAuthor:    batchedPostAuthor,
		postComments:  batchedPostComments,
		commentAuthor: batchedCommentAuthor,
		authorPosts:   batchedAuthorPosts,
	}
}

// batchedAuthorPosts is the worked example: queue a key, return a Deferred
// that reads the result once the loader has been dispatched. Three lines of
// shape, and the other three resolvers are the same shape.
//
// Notice what the Deferred buys: the resolver returns *without fetching*, so
// the executor can let every sibling resolver queue its key before anything
// touches the store.
func batchedAuthorPosts(ctx context.Context, parent any, args Args) (any, error) {
	l, err := loaders(ctx)
	if err != nil {
		return nil, err
	}
	res := l.Posts.Load(parent.(Author).ID)
	return Deferred(func() (any, error) {
		ps, err := res.Get()
		if err != nil {
			return nil, err
		}
		return List(applyFirst(ps, args)), nil
	}), nil
}

func batchedPostAuthor(ctx context.Context, parent any, _ Args) (any, error) {
	// TODO: load the post's author through l.Authors and return a Deferred
	// that yields it. An Author is not a list, so nothing needs List().
	return nil, nil
}

func batchedPostComments(ctx context.Context, parent any, args Args) (any, error) {
	// TODO: load the post's comments through l.Comments, honour `first` with
	// applyFirst, and return them through List().
	return nil, nil
}

func batchedCommentAuthor(ctx context.Context, parent any, _ Args) (any, error) {
	// TODO: the same as batchedPostAuthor, from a Comment. Loading an author
	// the post level already loaded must not queue a second key.
	return nil, nil
}
