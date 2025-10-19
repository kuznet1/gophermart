package errs

import (
	"net/http"
)

type HTTPError struct {
	msg  string
	code int
}

func NewHTTPError(msg string, code int) *HTTPError {
	return &HTTPError{
		msg:  msg,
		code: code,
	}
}

func (e HTTPError) Error() string {
	return e.msg
}

func (e HTTPError) Code() int {
	return e.code
}

var (
	ErrUserExists               = NewHTTPError("user is already exists", http.StatusConflict)
	ErrUserCredentials          = NewHTTPError("incorrect login or password", http.StatusUnauthorized)
	ErrInvalidOrderNum          = NewHTTPError("invalid order number", http.StatusUnprocessableEntity)
	ErrOrderUploadedByUser      = NewHTTPError("order uploaded by user", http.StatusOK)
	ErrOrderUploadedByOtherUser = NewHTTPError("order uploaded by other user", http.StatusConflict)
	ErrBalanceNotEnoughPoints   = NewHTTPError("not enough points", http.StatusPaymentRequired)
)
