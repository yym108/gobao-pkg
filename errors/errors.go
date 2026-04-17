package errors

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Code string

const (
	CodeInvalidArg  Code = "INVALID_ARGUMENT"
	CodeUnauth      Code = "UNAUTHENTICATED"
	CodeForbidden   Code = "PERMISSION_DENIED"
	CodeNotFound    Code = "NOT_FOUND"
	CodeConflict    Code = "CONFLICT"
	CodeExhausted   Code = "RESOURCE_EXHAUSTED"
	CodeInternal    Code = "INTERNAL"
	CodeUnavailable Code = "UNAVAILABLE"
)

type Error struct {
	Code Code
	Msg  string
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Msg)
	}
	return fmt.Sprintf("%s: %s: %s", e.Code, e.Msg, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(c Code, msg string) *Error {
	return &Error{Code: c, Msg: msg}
}

func Wrap(c Code, msg string, err error) *Error {
	return &Error{Code: c, Msg: msg, Err: err}
}

var grpcMap = map[Code]codes.Code{
	CodeInvalidArg:  codes.InvalidArgument,
	CodeUnauth:      codes.Unauthenticated,
	CodeForbidden:   codes.PermissionDenied,
	CodeNotFound:    codes.NotFound,
	CodeConflict:    codes.AlreadyExists,
	CodeExhausted:   codes.ResourceExhausted,
	CodeInternal:    codes.Internal,
	CodeUnavailable: codes.Unavailable,
}

func ToGRPCStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}
	if be, ok := err.(*Error); ok {
		if c, ok := grpcMap[be.Code]; ok {
			return status.New(c, be.Msg)
		}
		return status.New(codes.Internal, be.Msg)
	}
	return status.New(codes.Internal, "internal server error")
}
