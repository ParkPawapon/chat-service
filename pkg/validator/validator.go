package validator

import (
	"reflect"
	"strings"

	goValidator "github.com/go-playground/validator/v10"
)

type Validator struct {
	validator *goValidator.Validate
}

func New() *Validator {
	v := goValidator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return &Validator{validator: v}
}

func (v *Validator) Struct(value any) error {
	return v.validator.Struct(value)
}
