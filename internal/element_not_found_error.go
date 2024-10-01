package internal

import (
	"errors"
	"fmt"
)

type ElementNotFoundError struct {
	Selector string
}

func NewElementNotFoundError(selector string) *ElementNotFoundError {
	return &ElementNotFoundError{Selector: selector}
}

func (e ElementNotFoundError) Error() string {
	return fmt.Sprintf("element %s not found", e.Selector)
}

func (e ElementNotFoundError) Is(target error) bool {
	var t *ElementNotFoundError
	ok := errors.As(target, &t)
	return ok
}
