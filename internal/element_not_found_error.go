package internal

import (
	"errors"
	"fmt"
	"github.com/csr-ugra/avito-estate-parser/internal/selector"
)

type ElementNotFoundError struct {
	Selector string
}

func NewElementNotFoundError(selector selector.Selector) *ElementNotFoundError {
	return &ElementNotFoundError{Selector: string(selector)}
}

func (e ElementNotFoundError) Error() string {
	return fmt.Sprintf("element '%s' not found", e.Selector)
}

func (e ElementNotFoundError) Is(target error) bool {
	var t *ElementNotFoundError
	ok := errors.As(target, &t)
	return ok
}
