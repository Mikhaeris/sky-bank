package apperr

type Kind uint8

const (
	InvalidArgument Kind = iota + 1
	Unauthenticated
	NotFound
	RateLimited
)

type Error struct {
	kind    Kind
	reason  string
	message string
}

func New(kind Kind, reason, message string) *Error {
	return &Error{kind: kind, reason: reason, message: message}
}

func (e *Error) Error() string   { return e.message }
func (e *Error) Kind() Kind      { return e.kind }
func (e *Error) Reason() string  { return e.reason }
func (e *Error) Message() string { return e.message }
