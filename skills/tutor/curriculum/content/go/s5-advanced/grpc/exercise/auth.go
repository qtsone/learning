package tasks

import (
	"context"

	"google.golang.org/grpc"
)

// AuthUnaryInterceptor returns a server interceptor that rejects any unary
// RPC not carrying the metadata pair
//
//	authorization: Bearer <token>
//
// with codes.Unauthenticated. It runs before the handler, like HTTP
// middleware runs before a handler — same idea, different wrapping mechanism.
func AuthUnaryInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// TODO: read the incoming metadata (metadata.FromIncomingContext),
		// check the "authorization" value equals "Bearer <token>", and return
		// a codes.Unauthenticated status error when it doesn't.
		return handler(ctx, req)
	}
}
