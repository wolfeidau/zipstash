package middleware

import (
	"net/http"

	"github.com/awslabs/aws-lambda-go-api-proxy/core"
)

// RealIP returns a middleware that sets a http.Request's RemoteAddr to the results of parsing either the X-Forwarded-For or X-Real-IP header fields.
func APIGatewayV2(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()
		apigwctx, ok := core.GetAPIGatewayV2ContextFromContext(ctx)
		if ok {
			r.Header.Set("X-Forwarded-For", apigwctx.HTTP.SourceIP)
			r.Header.Set("X-RequestID", apigwctx.RequestID)
		}

		next.ServeHTTP(w, r)
	})
}
