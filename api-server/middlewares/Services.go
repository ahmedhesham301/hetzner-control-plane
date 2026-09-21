package middlewares

import (
	"encoding/json"
	"net/http"

	"github.com/ahmedhesham301/hetzner-control-plane/api-server/data"
	"github.com/gin-gonic/gin"
)

func ValidateParams() gin.HandlerFunc {
	return func(g *gin.Context) {
		var body json.RawMessage
		if err := g.ShouldBindJSON(&body); err != nil {
			g.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		params, err := data.ParseService(body)
		if err != nil {
			g.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		g.Set("params", params)
		g.Set("rawService", body)
		g.Next()
	}
}
