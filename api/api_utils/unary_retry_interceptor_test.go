package api_utils

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- noNeedRetry unit tests ---

func TestNoNeedRetry_NilError(t *testing.T) {
	assert.True(t, noNeedRetry(nil))
}

func TestNoNeedRetry_NonConnectError(t *testing.T) {
	assert.True(t, noNeedRetry(fmt.Errorf("plain error")))
}

func TestNoNeedRetry_RetriableCodes(t *testing.T) {
	retriable := []connect.Code{
		connect.CodeUnknown,
		connect.CodeInternal,
		connect.CodeUnavailable,
		connect.CodeAborted,
		connect.CodeResourceExhausted,
	}
	for _, code := range retriable {
		err := connect.NewError(code, fmt.Errorf("test"))
		assert.False(t, noNeedRetry(err), "code %v should be retriable", code)
	}
}

func TestNoNeedRetry_NonRetriableCodes(t *testing.T) {
	nonRetriable := []connect.Code{
		connect.CodeNotFound,
		connect.CodeInvalidArgument,
		connect.CodePermissionDenied,
		connect.CodeUnauthenticated,
		connect.CodeAlreadyExists,
		connect.CodeFailedPrecondition,
		connect.CodeUnimplemented,
	}
	for _, code := range nonRetriable {
		err := connect.NewError(code, fmt.Errorf("test"))
		assert.True(t, noNeedRetry(err), "code %v should not be retriable", code)
	}
}

// --- Interceptor behavior tests ---

// mockUnaryFunc creates a connect.UnaryFunc that returns errors from the given sequence.
// After the sequence is exhausted, it returns nil (success). callCount tracks invocations.
func mockUnaryFunc(callCount *atomic.Int32, errs ...error) connect.UnaryFunc {
	return func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		idx := int(callCount.Add(1)) - 1
		if idx < len(errs) {
			return nil, errs[idx]
		}
		return nil, nil
	}
}

func TestInterceptor_Success(t *testing.T) {
	var calls atomic.Int32
	next := mockUnaryFunc(&calls)

	interceptor := newUnaryRetryInterceptor(3, time.Millisecond, time.Millisecond)
	wrappedFunc := interceptor(next)

	_, err := wrappedFunc(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
}

func TestInterceptor_RetryThenSuccess(t *testing.T) {
	var calls atomic.Int32
	next := mockUnaryFunc(&calls,
		connect.NewError(connect.CodeUnavailable, fmt.Errorf("unavailable")),
	)

	interceptor := newUnaryRetryInterceptor(3, time.Millisecond, time.Millisecond)
	wrappedFunc := interceptor(next)

	_, err := wrappedFunc(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

func TestInterceptor_RetryExhausted(t *testing.T) {
	var calls atomic.Int32
	next := mockUnaryFunc(&calls,
		connect.NewError(connect.CodeInternal, fmt.Errorf("internal")),
		connect.NewError(connect.CodeInternal, fmt.Errorf("internal")),
		connect.NewError(connect.CodeInternal, fmt.Errorf("internal")),
	)

	interceptor := newUnaryRetryInterceptor(2, time.Millisecond, time.Millisecond)
	wrappedFunc := interceptor(next)

	_, err := wrappedFunc(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry failed after 2 attempts")
	assert.Equal(t, int32(3), calls.Load())
}

func TestInterceptor_NonRetriableCode(t *testing.T) {
	nonRetriable := []connect.Code{
		connect.CodeNotFound,
		connect.CodeInvalidArgument,
		connect.CodePermissionDenied,
		connect.CodeUnauthenticated,
		connect.CodeAlreadyExists,
		connect.CodeFailedPrecondition,
		connect.CodeUnimplemented,
	}

	for _, code := range nonRetriable {
		t.Run(code.String(), func(t *testing.T) {
			var calls atomic.Int32
			next := mockUnaryFunc(&calls,
				connect.NewError(code, fmt.Errorf("test")),
			)

			interceptor := newUnaryRetryInterceptor(3, time.Millisecond, time.Millisecond)
			wrappedFunc := interceptor(next)

			_, err := wrappedFunc(context.Background(), nil)
			require.Error(t, err)
			assert.Equal(t, code, connect.CodeOf(err))
			assert.Equal(t, int32(1), calls.Load(), "non-retriable code %v should not be retried", code)
		})
	}
}

func TestInterceptor_ResourceExhausted(t *testing.T) {
	var calls atomic.Int32
	next := mockUnaryFunc(&calls,
		connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limited")),
	)

	interceptor := newUnaryRetryInterceptor(3, time.Millisecond, time.Millisecond)
	wrappedFunc := interceptor(next)

	_, err := wrappedFunc(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}
