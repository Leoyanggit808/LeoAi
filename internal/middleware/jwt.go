package middleware

import (
	"LeoAi/config"
	"LeoAi/internal/util"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		//获取 Authorization 请求头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供token"})
			c.Abort() //// 重要！终止后续执行
			return
		}
		//去掉 "Bearer " 前缀
		//strings.TrimPrefix(s, prefix) 是 Go 标准库 strings 包里的函数。
		//如果字符串 s 以 prefix 开头，就把这个前缀去掉；否则原样返回。
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		//解析 Token
		claims, err := util.ParseToken(tokenString, cfg.JWTSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			c.Abort()
			return
		}
		//把用户信息存入 Context，方便后续 Handler 使用
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
