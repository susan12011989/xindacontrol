package result

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GAuthErr(c *gin.Context) {
	c.Abort()
	c.JSON(http.StatusUnauthorized, nil)
}

func GOK(c *gin.Context, data any) {
	c.Abort()
	GResult(c, 200, data)
}

func GErr(c *gin.Context, err error) {
	c.Abort()
	GResult(c, 400, nil, err.Error())
}

func GParamErr(c *gin.Context, err error) {
	c.Abort()
	GResult(c, 601, nil, err.Error())
}

func GResult(c *gin.Context, code int, data any, msg ...string) {
	c.Abort()
	var tmpMsg string
	if len(msg) > 0 {
		tmpMsg = msg[0]
	}
	obj := gin.H{
		"code":    code,
		"data":    data,
		"message": tmpMsg,
	}
	c.JSON(http.StatusOK, obj)
}

// GStream 设置流式响应头部并准备SSE流
func GStream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Status(http.StatusOK)
}

// GStreamData 发送流式数据
func GStreamData(c *gin.Context, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		// 发生错误时发送错误信息
		c.SSEvent("error", fmt.Sprintf("序列化错误: %v", err))
		return
	}

	// 使用SSE格式发送数据
	c.SSEvent("data", string(jsonData))
	c.Writer.Flush()
}

// GStreamEnd 结束流式响应
func GStreamEnd(c *gin.Context, success bool, message string) {
	response := gin.H{
		"success": success,
		"message": message,
	}
	c.SSEvent("end", response)
	c.Writer.Flush()
}

func MarshalJson(s any) string {
	bytes, _ := json.Marshal(s)
	return string(bytes)
}

func UnAuthorization(ctx *gin.Context, reason string) {
	ctx.Abort()
	ctx.JSON(http.StatusUnauthorized, map[string]any{
		"reason": reason,
	})
}
