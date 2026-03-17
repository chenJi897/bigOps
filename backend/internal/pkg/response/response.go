// Package response 提供统一的 HTTP API 响应格式。
// 所有 API 接口均通过本包返回标准化的 JSON 响应，确保前后端约定一致。
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 标准 API 响应结构体。
// Code 为 0 表示成功，非零为业务错误码。
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData 分页数据结构体，用于列表类接口的响应。
type PageData struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"` // 总记录数
	Page  int         `json:"page"`  // 当前页码
	Size  int         `json:"size"`  // 每页条数
}

// Success 返回成功响应（code=0）。
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 返回带自定义消息的成功响应。
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error 返回错误响应，code 为业务错误码。
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithData 返回带附加数据的错误响应（如字段级校验错误详情）。
func ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// BadRequest 返回 400 参数错误响应。
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized 返回 401 未认证响应。
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

// Forbidden 返回 403 无权限响应。
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

// NotFound 返回 404 资源未找到响应。
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

// InternalServerError 返回 500 服务器内部错误响应。
func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}

// Page 返回分页列表响应。
func Page(c *gin.Context, list interface{}, total int64, page, size int) {
	Success(c, PageData{
		List:  list,
		Total: total,
		Page:  page,
		Size:  size,
	})
}
