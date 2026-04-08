// Package mempalace provides a public Go API for MemPalace operations.
package mempalace

// ErrorCode represents a category of error.
type ErrorCode string

const (
	// ErrConfigLoad indicates configuration loading failed.
	ErrConfigLoad ErrorCode = "config_load"
	// ErrStoreInit indicates store initialization failed.
	ErrStoreInit ErrorCode = "store_init"
	// ErrEmbedding indicates embedding generation failed.
	ErrEmbedding ErrorCode = "embedding"
	// ErrSearch indicates search operation failed.
	ErrSearch ErrorCode = "search"
	// ErrAdd indicates add operation failed.
	ErrAdd ErrorCode = "add"
	// ErrMine indicates mining operation failed.
	ErrMine ErrorCode = "mine"
	// ErrDuplicate indicates duplicate content detected.
	ErrDuplicate ErrorCode = "duplicate"
	// ErrNotFound indicates resource not found.
	ErrNotFound ErrorCode = "not_found"
	// ErrInvalidInput indicates invalid input provided.
	ErrInvalidInput ErrorCode = "invalid_input"
	// ErrClosed indicates operation on closed palace.
	ErrClosed ErrorCode = "closed"
)

// Error represents a MemPalace error with code and context.
type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is checks if an error matches a specific error code.
func Is(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}
	var e *Error
	if As(err, &e) {
		return e.Code == code
	}
	return false
}

// As checks if an error can be assigned to target.
func As(err error, target **Error) bool {
	if err == nil {
		return false
	}
	for {
		if e, ok := err.(*Error); ok {
			*target = e
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
			if err == nil {
				return false
			}
		} else {
			return false
		}
	}
}

// NewError creates a new Error with the given code and message.
func NewError(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}