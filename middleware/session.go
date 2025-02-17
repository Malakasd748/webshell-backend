package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetSessionKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从头部获取集群和用户名
		username, cluster, sessionKey := c.GetHeader("username"), c.GetHeader("cluster"), c.GetHeader("sessionKey")
		if sessionKey != "" {
			c.Set("sessionKey", sessionKey)
			return
		}
		if cluster == "" {
			cluster = "default"
		}
		if username != "" && cluster != "" {
			c.Set("sessionKey", fmt.Sprintf("%s_%s", cluster, username))
			return
		}

		// 根据请求方法获取集群和用户名
		switch c.Request.Method {
		case http.MethodGet:
			username = c.Query("username")
			cluster = c.Query("cluster")
			sessionKey = c.Query("sessionKey")
			if cluster == "" {
				cluster = "default"
			}
		case http.MethodPost:
			contentType := c.GetHeader("Content-Type")
			switch contentType {
			case "application/json":
				var data map[string]string
				if err := c.ShouldBindJSON(&data); err != nil {
					c.AbortWithError(http.StatusBadRequest, err)
					return
				}
				username = data["username"]
				cluster = data["cluster"]
				sessionKey = data["sessionKey"]
				if cluster == "" {
					cluster = "default"
				}
			case "application/x-www-form-urlencoded", "multipart/form-data":
				username, _ = c.GetPostForm("username")
				cluster, _ = c.GetPostForm("cluster")
				sessionKey, _ = c.GetPostForm("sessionKey")
				if cluster == "" {
					cluster = "default"
				}
			default:
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("unsupported content type: %s", contentType))
			}
		default:
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("unsupported method: %s", c.Request.Method))
		}

		if sessionKey != "" {
			c.Set("sessionKey", sessionKey)
			return
		}
		// 生成并返回键
		if username != "" && cluster != "" {
			c.Set("sessionKey", fmt.Sprintf("%s_%s", cluster, username))
			return
		}
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("failed to retrieve username or cluster"))
	}
}
