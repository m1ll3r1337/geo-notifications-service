package errs

import (
	"errors"
	"fmt"
)

type Kind string

const (
	KindInvalid      Kind = "invalid"
	KindNotFound     Kind = "not_found"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindConflict     Kind = "conflict"
	KindInternal     Kind = "internal"
)

type Error struct {
	Kind   Kind
	Code   string
	Op     string
	Msg    string
	Fields map[string]string
	Err    error
}

func (e *Error) Error() string {
	base := e.Msg
	if base == "" && e.Err != nil {
		base = e.Err.Error()
	}
	if base == "" {
		base = string(e.Kind)
	}
	if e.Op != "" {
		return fmt.Sprintf("%s: %s", e.Op, base)
	}
	return base
}

func (e *Error) Unwrap() error { return e.Err }

func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

func E(kind Kind, code, op, msg string, fields map[string]string, err error) *Error {
	return &Error{
		Kind:   kind,
		Code:   code,
		Op:     op,
		Msg:    msg,
		Fields: fields,
		Err:    err,
	}
}

// Wrap preserves Kind/Code/Fields if err is already *errs.Error, and sets a new Op.
func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	if e, ok := As(err); ok {
		return &Error{
			Kind:   e.Kind,
			Code:   e.Code,
			Op:     op,
			Msg:    e.Msg,
			Fields: e.Fields,
			Err:    err,
		}
	}
	return &Error{
		Kind: KindInternal,
		Code: "INTERNAL",
		Op:   op,
		Msg:  "internal error",
		Err:  err,
	}
}
