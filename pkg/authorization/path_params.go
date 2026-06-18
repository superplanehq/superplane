package authorization

import (
	"context"
	"strings"
)

type pathParamsContextKey struct{}

func WithPathParams(ctx context.Context, pathParams map[string]string) context.Context {
	if len(pathParams) == 0 {
		return ctx
	}

	copied := make(map[string]string, len(pathParams))
	for key, value := range pathParams {
		copied[key] = value
	}

	return context.WithValue(ctx, pathParamsContextKey{}, copied)
}

func PathParamsFromContext(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}

	pathParams, ok := ctx.Value(pathParamsContextKey{}).(map[string]string)
	if !ok || len(pathParams) == 0 {
		return nil
	}

	return pathParams
}

func resourceIDsFromPathParams(pathParams map[string]string, keys []string) []string {
	if len(pathParams) == 0 || len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		value := strings.TrimSpace(pathParams[key])
		if value != "" {
			return []string{value}
		}
	}

	return nil
}
