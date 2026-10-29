package fail

import (
	"errors"
	"fmt"
)

// Error is a client-safe failure with an HTTP status.
type Error struct {
	Status int
	Msg    string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Msg
}

func New(status int, msg string) *Error {
	return &Error{Status: status, Msg: msg}
}

func Newf(status int, format string, args ...any) *Error {
	return &Error{Status: status, Msg: fmt.Sprintf(format, args...)}
}

// IsStatus reports whether err is a client error with the given HTTP status.
func IsStatus(err error, status int) bool {
	var fe *Error
	return errors.As(err, &fe) && fe.Status == status
}
