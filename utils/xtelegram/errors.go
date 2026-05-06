package xtelegram

import (
	"fmt"

	"github.com/xbaseio/xbase/xerrors"
)

var (
	ErrorForbidden       = xerrors.New("forbidden")
	ErrorBadRequest      = xerrors.New("bad request")
	ErrorUnauthorized    = xerrors.New("unauthorized")
	ErrorTooManyRequests = xerrors.New("too many requests")
	ErrorNotFound        = xerrors.New("not found")
	ErrorConflict        = xerrors.New("conflict")
)

type TooManyRequestsError struct {
	Message    string
	RetryAfter int
}

func (e *TooManyRequestsError) Error() string {
	return fmt.Sprintf("%s: retry_after %d", e.Message, e.RetryAfter)
}

func IsTooManyRequestsError(err error) bool {
	_, ok := err.(*TooManyRequestsError)
	return ok
}

type MigrateError struct {
	Message         string
	MigrateToChatID int
}

func (e *MigrateError) Error() string {
	return fmt.Sprintf("%s: migrate_to_chat_id %d", e.Message, e.MigrateToChatID)
}

func IsMigrateError(err error) bool {
	_, ok := err.(*MigrateError)
	return ok
}
