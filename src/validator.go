package main

import (
	"reflect"
	"regexp"
	"sync"

	"gopkg.in/go-playground/validator.v8"

	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v9"
)

var regCommandName = regexp.MustCompile(`https://?[\da-z.-]+\.(retailcrm\.(ru|pro)|ecomlogic\.com|simlachat\.com)`)

type defaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

var _ binding.StructValidator = &defaultValidator{}

func (v *defaultValidator) ValidateStruct(obj interface{}) error {

	if kindOfData(obj) == reflect.Struct {
		v.lazyinit()
		if err := v.validate.Struct(obj); err != nil {
			return error(err)
		}
	}

	return nil
}

func (v *defaultValidator) Engine() interface{} {
	v.lazyinit()
	return v.validate
}

func (v *defaultValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName("binding")
	})
}

func kindOfData(data interface{}) reflect.Kind {
	value := reflect.ValueOf(data)
	valueType := value.Kind()

	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	return valueType
}

func setValidation() {
	binding.Validator = new(defaultValidator)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("validatecrmurl", validateCrmURL)
	}
}

func validateCrmURL(field validator.FieldLevel) bool {
	return regCommandName.Match([]byte(field.Field().Interface().(string)))
}
