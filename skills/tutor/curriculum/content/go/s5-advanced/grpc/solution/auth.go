package tasks

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthUnaryInterceptor returns a server interceptor that rejects any unary
// RPC not carrying the metadata pair
//
//	authorization: Bearer <token>
//
// with codes.Unauthenticated. It runs before the handler, like HTTP
// middleware runs before a handler — same idea, different wrapping mechanism.
func AuthUnaryInterceptor(token string) grpc.UnaryServerInterceptor {
	want := "Bearer " + token
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		vals := md.Get("authorization")
		if len(vals) == 0 || vals[0] != want {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid bearer token")
		}
		return handler(ctx, req)
	}
}
