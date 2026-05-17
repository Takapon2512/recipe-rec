package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthResponse struct {
	Status  string    `json:"status"`
	Version int       `json:"id"`
	Time    time.Time `json:"time"`
}

func NewHealthHandler() gin.HandlerFunc {
	now := time.Now()

	return func(c *gin.Context) {
		c.JSON(
			http.StatusOK,
			gin.H{
				"data": HealthResponse{
					Status:  "OK",
					Version: 1,
					Time:    now,
				},
			},
		)
	}
}
