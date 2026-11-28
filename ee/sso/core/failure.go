package core

import "fmt"

const (
	CodeNotAvailable    = "sso_not_available"
	CodeFailed          = "sso_failed"
	CodeAccessDenied    = "sso_access_denied"
	CodeAccountConflict = "sso_account_conflict"
	CodeSeatLimit       = "sso_seat_limit"
	CodeNotLicensed     = "sso_not_licensed"
	CodeAccountBlocked  = "account_blocked"
)

// Failure is a sign-in result the browser can show.
type Failure struct {
	Code string
}

func (f *Failure) Error() string {
	if f == nil {
		return ""
	}
	return f.Code
}

func Fail(code string) error {
	return &Failure{Code: code}
}

func CodeOf(err error) string {
	if err == nil {
		return ""
	}
	if failure, ok := err.(*Failure); ok {
		return failure.Code
	}
	return ""
}

func WrapFailed(err error) error {
	if err == nil {
		return nil
	}
	if CodeOf(err) != "" {
		return err
	}
	return fmt.Errorf("%w", Fail(CodeFailed))
}
