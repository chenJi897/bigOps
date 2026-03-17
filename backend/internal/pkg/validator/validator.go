// Package validator 封装 go-playground/validator，提供请求参数校验功能。
// 通过 Gin 的 binding 引擎自动关联，并将校验错误翻译为可读的中文/英文提示。
package validator

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// validate 与 Gin 绑定引擎共享的校验器实例。
var validate *validator.Validate

func init() {
	// 从 Gin 的 binding 引擎获取底层 validator 实例，
	// 这样通过 ShouldBind 系列方法绑定的结构体会自动使用相同的校验规则。
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		validate = v
	}
}

// Validate 对结构体进行校验，返回校验错误（如有）。
func Validate(obj interface{}) error {
	if validate == nil {
		return fmt.Errorf("validator not initialized")
	}
	return validate.Struct(obj)
}

// TranslateError 将校验错误翻译为可读的错误消息字符串。
// 多个字段错误以分号分隔。非校验类错误原样返回。
func TranslateError(err error) string {
	if err == nil {
		return ""
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}

	var messages []string
	for _, e := range validationErrors {
		messages = append(messages, translateFieldError(e))
	}

	return strings.Join(messages, "; ")
}

// translateFieldError 将单个字段的校验错误翻译为可读消息。
func translateFieldError(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, e.Param())
	case "len":
		return fmt.Sprintf("%s must be %s characters", field, e.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", field, e.Param())
	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}
