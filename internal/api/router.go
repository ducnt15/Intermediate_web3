package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterApi(router *gin.Engine) error {
	// Register tracking routes
	trackingGroup := router.Group("/tracking")
	{
		trackingGroup.GET("", GetTracking)
		trackingGroup.GET("/:search", GetTrackingByKey)
		trackingGroup.DELETE("/:transaction", DeleteTrackingTransaction)
	}
	return nil
}
