package auth

import "context"

// authKey is unexported, so nothing outside this package can plant a User in a
// request context: the only way in is through the middleware, which is the
// only code that has verified anything.
type authKey struct{}

type authContext struct {
	user    User
	session Session
}

// withAuth returns a copy of ctx carrying the authenticated identity.
func withAuth(ctx context.Context, user User, session Session) context.Context {
	return context.WithValue(ctx, authKey{}, authContext{user: user, session: session})
}

// UserFromContext returns the authenticated user. Handlers behind
// Service.Authenticate can rely on ok being true; anywhere else it is false.
func UserFromContext(ctx context.Context) (User, bool) {
	a, ok := ctx.Value(authKey{}).(authContext)
	return a.user, ok
}

// SessionFromContext returns the session the request authenticated with, so a
// handler never has to re-parse the cookie the middleware already validated.
func SessionFromContext(ctx context.Context) (Session, bool) {
	a, ok := ctx.Value(authKey{}).(authContext)
	return a.session, ok
}
