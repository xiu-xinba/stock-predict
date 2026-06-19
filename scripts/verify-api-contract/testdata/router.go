package fixture

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func routes(engine *gin.Engine, handler gin.HandlerFunc) {
	v1 := engine.Group("/api/v1")
	v1.GET(
		"/multiline/:code",
		handler,
	)
	v1.Handle(
		http.MethodPost,
		"/handled",
		handler,
	)
}
