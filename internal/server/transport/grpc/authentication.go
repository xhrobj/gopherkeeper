package grpcserver

import (
	"context"
	"strings"

	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const authorizationSchemeBearer = "Bearer"

type userIDContextKey struct{}

func authenticationUnaryInterceptor(validator TokenValidator) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if isPublicMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		token, ok := bearerTokenFromMetadata(ctx)
		if !ok || validator == nil {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}

		userID, err := validator.Validate(ctx, token)
		if err != nil || userID <= 0 {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}

		return handler(context.WithValue(ctx, userIDContextKey{}, userID), req)
	}
}

func isPublicMethod(fullMethod string) bool {
	switch fullMethod {
	case gopherkeeperpb.AuthService_Register_FullMethodName,
		gopherkeeperpb.AuthService_Login_FullMethodName,
		healthpb.Health_Check_FullMethodName,
		healthpb.Health_List_FullMethodName:
		return true
	default:
		return false
	}
}

func bearerTokenFromMetadata(ctx context.Context) (string, bool) {
	incoming, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	values := incoming.Get("authorization")
	if len(values) != 1 {
		return "", false
	}

	fields := strings.Fields(values[0])
	if len(fields) != 2 || !strings.EqualFold(fields[0], authorizationSchemeBearer) {
		return "", false
	}

	return fields[1], true
}

func userIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(int64)
	return userID, ok && userID > 0
}
