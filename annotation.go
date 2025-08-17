package foxtimeout

import (
	"time"

	"github.com/tigerwill90/fox"
)

type key struct{}

var ctxKey key

// After returns a RouteOption that sets a custom timeout duration for a specific route.
// This allows individual routes to have different timeout values than the global timeout.
func After(dt time.Duration) fox.RouteOption {
	return fox.WithAnnotation(ctxKey, dt)
}

// None returns a RouteOption that disables the timeout for a specific route.
// This is useful for long-running operations like file uploads or SSE endpoints.
func None() fox.RouteOption {
	return fox.WithAnnotation(ctxKey, time.Duration(0))
}

func unwrapRouteTimeout(r *fox.Route) (time.Duration, bool) {
	dt := r.Annotation(ctxKey)
	if dt != nil {
		return dt.(time.Duration), true
	}
	return 0, false
}
