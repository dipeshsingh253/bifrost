package http

import "github.com/gin-gonic/gin"

func Success(data any) gin.H {
	return gin.H{
		"success": true,
		"data":    data,
	}
}

func Error(message string, code string, details any) gin.H {
	return gin.H{
		"success": false,
		"error": gin.H{
			"message": message,
			"code":    code,
			"details": details,
		},
	}
}
