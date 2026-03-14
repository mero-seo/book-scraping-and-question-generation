package services

// ErrNotFound indicates a resource was not found.
type ErrNotFound struct {
	Message string
}

func (e ErrNotFound) Error() string { return e.Message }

// ErrConflict indicates a duplicate resource.
type ErrConflict struct {
	Message string
}

func (e ErrConflict) Error() string { return e.Message }

// ErrUnauthorized indicates invalid credentials.
type ErrUnauthorized struct {
	Message string
}

func (e ErrUnauthorized) Error() string { return e.Message }

// ErrValidation indicates invalid input.
type ErrValidation struct {
	Message string
}

func (e ErrValidation) Error() string { return e.Message }

// ErrForbidden indicates insufficient permissions.
type ErrForbidden struct {
	Message string
}

func (e ErrForbidden) Error() string { return e.Message }
