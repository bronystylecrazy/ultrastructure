package web

import "github.com/go-playground/validator/v10"

type FiberValidator struct {
	validate *validator.Validate
}

func NewFiberValidator() *FiberValidator {
	return &FiberValidator{
		validate: validator.New(),
	}
}

func (v *FiberValidator) Validate(out any) error {
	return v.validate.Struct(out)
}
