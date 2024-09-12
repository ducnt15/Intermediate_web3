package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterApi(router *gin.Engine) error {
	// Register service routes
	trackingGroup := router.Group("/service")
	{
		trackingGroup.GET("", GetTracking)
		trackingGroup.GET("/:search", GetTrackingByKey)
		trackingGroup.DELETE("/:transaction", DeleteTrackingTransaction)
	}
	return nil
}
