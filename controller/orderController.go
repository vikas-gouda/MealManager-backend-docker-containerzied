package controller

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vikas-gouda/go-restraunt-mangement/database"
	"github.com/vikas-gouda/go-restraunt-mangement/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var orderCollection *mongo.Collection = database.OpenCollection(database.Client, "order")

func GetOrders() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		result, err := orderCollection.Find(context.Background(), bson.M{})
		defer cancel()

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while listing the order"})
			return
		}

		var allOrder []bson.M
		if err = result.All(c, &allOrder); err != nil {
			log.Fatal(err.Error())
			return
		}

		ctx.JSON(http.StatusOK, allOrder[0])
	}
}

func GetOrder() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		orderId := ctx.Param("order_id")

		var order models.Order

		err := orderCollection.FindOne(c, bson.M{"order_id": orderId}).Decode(&order)
		defer cancel()

		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, order)

	}
}

func CreateOrder() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var order models.Order
		var table models.Order

		if err := ctx.BindJSON(&order); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(order)
		if validationErr != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": validationErr.Error()})
			return
		}

		err := tableCollection.FindOne(c, bson.M{"table_id": order.Table_id}).Decode(&table)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Table not found"})
			return
		}

		order.Order_Date, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		order.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		order.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		order.ID = primitive.NewObjectID()
		order.Order_id = order.ID.Hex()

		result, insertErr := orderCollection.InsertOne(c, order)
		if insertErr != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while inserting"})
			return
		}
		defer cancel()

		ctx.JSON(http.StatusOK, result)
	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var order models.Order
		var table models.Table

		if err := ctx.BindJSON(&order); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		orderId := ctx.Param("order_id")
		tableId := ctx.Param("table_id")

		err := tableCollection.FindOne(c, bson.M{"table_id": tableId}).Decode(&table)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Table not found"})
			return
		}

		var updateObj primitive.D

		order.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", order.Updated_at})

		upsert := true
		filter := bson.M{"order_id": orderId}
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		result, err := orderCollection.UpdateOne(c, filter, bson.D{
			{"$set", updateObj},
		}, &opt)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the order"})
			return
		}

		defer cancel()
		ctx.JSON(http.StatusOK, result)
	}

}

func OrderItemOrderCreator(order models.Order) string {
	var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	order.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	order.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	order.ID = primitive.NewObjectID()
	order.Order_id = order.ID.Hex()

	orderCollection.InsertOne(c, order)
	defer cancel()

	return order.Order_id
}
