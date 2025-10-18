package middleware

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	UserIDKey contextKey = "uid"
	EmailKey  contextKey = "email"
)

func JWTAuthInterceptor(secret string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeaders, ok := md["authorization"]
		if !ok || len(authHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		tokenStr := strings.TrimPrefix(authHeaders[0], "Bearer ")
		if tokenStr == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header")
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, status.Error(codes.Unauthenticated, "unexpected signing method")
			}

			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if uid, ok := claims["uid"].(float64); ok {
				ctx = context.WithValue(ctx, UserIDKey, int64(uid))
			}
			if email, ok := claims["email"].(string); ok {
				ctx = context.WithValue(ctx, EmailKey, email)
			}
		}

		return handler(ctx, req)
	}
}
