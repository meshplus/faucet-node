package global

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Msg  string `json:"msg"`
	Data string `json:"txHash"`
	Code int    `json:"code"`
}

func Result(res *Response, c *gin.Context) {
	// 开始时间
	c.JSON(http.StatusOK, res)
}

func Success(data string) *Response {
	return &Response{
		Code: SUCCESS,
		Data: data,
		Msg:  SUCCESSMsg,
	}
}

func Fail(code int, Msg string) *Response {
	return &Response{
		Code: code,
		Msg:  Msg,
	}
}
