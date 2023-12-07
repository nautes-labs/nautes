package auth

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

type AuthType string

const Authorization = "Authorization"
const BearerToken AuthType = "token"
const Oauth2 AuthType = "oauth2"

func SetTokenInContext() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				header := tr.RequestHeader()
				bearerToken := header.Get(Authorization)
				token := strings.TrimSpace(strings.Replace(bearerToken, "Bearer", "", 1))

				// Threr are two ways obtain token:
				// 1、Use Nautes clinet call to get the token in the request header.
				// 2、Parse the Context and obtain token in HTTP request.
				if token != "" {
					ctx = context.WithValue(ctx, BearerToken, token)
					authType := header.Get("AuthType")
					ctx = context.WithValue(ctx, Oauth2, authType)
				} else {
					if md, ok := metadata.FromServerContext(ctx); ok {
						token := md.Get(Authorization)
						ctx = context.WithValue(ctx, BearerToken, token)
					}
				}
			}

			return handler(ctx, req)
		}
	}
}
