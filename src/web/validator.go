package web

import "github.com/go-playground/validator/v10"

type Validator interface {
	Struct(value any) error
}

type defaultValidator struct {
	validate *validator.Validate
}

func NewValidator() Validator {
	return &defaultValidator{
		validate: validator.New(),
	}
}

func (v *defaultValidator) Struct(value any) error {
	return v.validate.Struct(value)
}
