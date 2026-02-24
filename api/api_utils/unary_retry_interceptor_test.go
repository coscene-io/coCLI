package api_utils

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
)

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
