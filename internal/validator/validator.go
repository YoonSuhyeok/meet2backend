package validator

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func RegisterValidators() {
	var startTimeRe = regexp.MustCompile(`^([01]\d|2[0-3]):[03]0$`)
	var endTimeRe = regexp.MustCompile(`^(([01]\d|2[0-3]):[03]0|24:00)$`)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("start_time", func(fl validator.FieldLevel) bool {
			return startTimeRe.MatchString(fl.Field().String())
		})
		_ = v.RegisterValidation("end_time", func(fl validator.FieldLevel) bool {
			return endTimeRe.MatchString(fl.Field().String())
		})
	}
}
