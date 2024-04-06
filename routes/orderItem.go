package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vikas-gouda/go-restraunt-mangement/controller"
)

func OrderItemRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.GET("/orderItems", controller.GetOrderItems())
	incomingRoutes.GET("orderItems/:orderItems_id", controller.GetOrderItem())
	incomingRoutes.GET("/orderItems-order/:order_id", controller.GetOrderItemsByOrder())
	incomingRoutes.POST("/orderItems", controller.CreateOrderItem())
	incomingRoutes.PATCH("/orderItems/:orderItem_id", controller.UpdateOrderItem())
}
