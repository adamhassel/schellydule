package contx

import (
	"context"
	"net/http"
	"strconv"
)

type ctxkey int

const pretend ctxkey = iota

// ProcessCommon processes the request for common parameters, and returns a context with values
func ProcessCommon(r *http.Request) context.Context {
	query := r.URL.Query()
	p, _ := strconv.ParseBool(query.Get("pretend"))
	ctx := r.Context()
	ctx = context.WithValue(ctx, pretend, p)
	r = r.WithContext(ctx)
	return ctx
}

func Pretend(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v := ctx.Value(pretend)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
