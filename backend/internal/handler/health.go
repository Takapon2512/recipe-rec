package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

func NewHealthHandler() gin.HandlerFunc {

	return func(c *gin.Context) {
		c.JSON(
			http.StatusOK,
			gin.H{
				"data": HealthResponse{
					Status:    "ok",
					Version:   "1.0.0",
					Timestamp: time.Now(),
				},
			},
		)
	}
}
