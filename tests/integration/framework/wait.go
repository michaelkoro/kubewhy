package framework

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// WaitFor polls every 2 seconds until cond returns true or the timeout
// expires. fetch is called each tick to retrieve the latest state of the
// resource; the result is passed to cond.
//
// fetch should return (nil, nil) while the resource is not yet available so
// that WaitFor keeps polling rather than failing immediately.
//
// This is generic — T can be *corev1.Pod, *appsv1.Deployment, etc.
func WaitFor[T any](
	t *testing.T,
	timeout time.Duration,
	fetch func(ctx context.Context) (T, error),
	cond func(T) bool,
	describe ...func(T) string,
) T {
	t.Helper()

	deadline := time.After(timeout)
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	var zero T
	var last T
	for {
		select {
		case <-deadline:
			if len(describe) > 0 {
				t.Fatalf("timed out after %s waiting for condition; last observed state: %s", timeout, describe[0](last))
			} else {
				t.Fatalf("timed out after %s waiting for condition", timeout)
			}
			return zero
		case <-tick.C:
			obj, err := fetch(context.Background())
			if err != nil {
				t.Logf("WaitFor: fetch error (will retry): %v", err)
				continue
			}
			last = obj
			if cond(obj) {
				return obj
			}
			if len(describe) > 0 {
				fmt.Fprintf(os.Stderr, "\t    --- %s: %s\n", t.Name(), describe[0](obj))
			}
		}
	}
}
