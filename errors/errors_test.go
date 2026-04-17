package errors_test

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/yym/gobao-pkg/errors"
)

func TestNew_basic(t *testing.T) {
	err := errors.New(errors.CodeNotFound, "user not found")

	require.NotNil(t, err)
	assert.Equal(t, errors.CodeNotFound, err.Code)
	assert.Equal(t, "user not found", err.Msg)
	assert.Nil(t, err.Err)
	assert.Equal(t, "NOT_FOUND: user not found", err.Error())
}

func TestWrap_preservesUnderlying(t *testing.T) {
	base := stderrors.New("sql: no rows")
	err := errors.Wrap(errors.CodeNotFound, "load user", base)

	assert.Equal(t, errors.CodeNotFound, err.Code)
	assert.Equal(t, "load user", err.Msg)
	assert.Same(t, base, err.Unwrap())
	assert.True(t, stderrors.Is(err, base))
	assert.Equal(t, "NOT_FOUND: load user: sql: no rows", err.Error())
}

func TestToGRPCStatus_mapsAllCodes(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want codes.Code
	}{
		{"nil", nil, codes.OK},
		{"invalid_arg", errors.New(errors.CodeInvalidArg, "x"), codes.InvalidArgument},
		{"unauth", errors.New(errors.CodeUnauth, "x"), codes.Unauthenticated},
		{"forbidden", errors.New(errors.CodeForbidden, "x"), codes.PermissionDenied},
		{"not_found", errors.New(errors.CodeNotFound, "x"), codes.NotFound},
		{"conflict", errors.New(errors.CodeConflict, "x"), codes.AlreadyExists},
		{"exhausted", errors.New(errors.CodeExhausted, "x"), codes.ResourceExhausted},
		{"internal", errors.New(errors.CodeInternal, "x"), codes.Internal},
		{"unavailable", errors.New(errors.CodeUnavailable, "x"), codes.Unavailable},
		{"non_business_err", stderrors.New("raw"), codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := errors.ToGRPCStatus(tc.in)
			assert.Equal(t, tc.want, st.Code())
		})
	}
}
