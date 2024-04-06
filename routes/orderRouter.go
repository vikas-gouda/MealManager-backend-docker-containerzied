package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vikas-gouda/go-restraunt-mangement/controller"
)

func OrderRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.GET("/orders", controller.GetOrders())
	incomingRoutes.GET("orders/:order_id", controller.GetOrder())
	incomingRoutes.POST("/orders", controller.CreateOrder())
	incomingRoutes.PATCH("/orders/:order_id", controller.UpdateOrder())
}
