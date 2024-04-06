package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vikas-gouda/go-restraunt-mangement/controller"
)

func TableRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.GET("/tables", controller.GetTables())
	incomingRoutes.GET("tables/:table_id", controller.GetTable())
	incomingRoutes.POST("/tables", controller.CreateTable())
	incomingRoutes.PATCH("/tables/:table_id", controller.UpdateTable())
}
