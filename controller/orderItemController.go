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

type OrderItemPack struct {
	Table_id   string
	Oder_items []models.OrderItem
}

var orderItemCollection = database.OpenCollection(database.Client, "orderItem")

func GetOrderItems() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		result, err := orderCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var allOrderItems []bson.M
		if err := result.All(c, &allOrderItems); err != nil {
			log.Fatal(err)
			return
		}

		ctx.JSON(http.StatusOK, allOrderItems)

	}
}

func GetOrderItemsByOrder() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		orderId := ctx.Param("order_id")

		allOrderItems, err := ItemsByOrder(orderId)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error occured while listing the order items by order id"})
			return
		}

		ctx.JSON(http.StatusOK, allOrderItems)
	}
}

func ItemsByOrder(id string) (OrderItems []primitive.M, err error) {
	var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

	matchStage := bson.D{{"$match", bson.D{{"order_id", id}}}}
	lookupStage := bson.D{{"$lookup", bson.D{{"from", "food"}, {"localField", "food_id"}, {"foreignField", "food_id"}, {"as", "food"}}}}
	unwindStage := bson.D{{"$unwind", bson.D{{"path", "$food"}, {"preserveNullAndEmptyArrays", true}}}}

	lookupOrderStage := bson.D{{"$lookup", bson.D{{"from", "order"}, {"localField", "order_id"}, {"foreignField", "order_id"}, {"as", "order"}}}}
	unwindOrderStage := bson.D{{"$unwind", bson.D{{"path", "$order"}, {"preserveNullAndEmptyArrays", true}}}}

	lookupTableStage := bson.D{{"$lookup", bson.D{{"from", "table"}, {"localField", "order.table_id"}, {"foreignField", "table_id"}, {"as", "table"}}}}
	unwindTableStage := bson.D{{"$unwind", bson.D{{"path", "$table"}, {"preserveNullAndEmptyArrays", true}}}}

	projectStage := bson.D{
		{
			"$project", bson.D{
				{"id", 0},
				{"amount", "$food.price"},
				{"total_count", 1},
				{"food_name", "$food.name"},
				{"food_image", "$food.food_image"},
				{"table_number", "$table.table_number"},
				{"table_id", "$table.table_id"},
				{"order_id", "$order.order_id"},
				{"price", "$food.price"},
				{"quantity", 1},
			},
		},
	}

	groupStage := bson.D{
		{"$group", bson.D{{"$group", bson.D{{"_id", bson.D{{"order_id", "$order_id "}, {"table_id", "$table_id"}, {"table_number", "$table_number"}}}, {"payment_due", bson.D{{"$sum", "$amount"}}}, {"total_count", bson.D{{"$sum", 1}}}, {"order_items", bson.D{{}}}}}}},
	}

	projectStage2 := bson.D{
		{"$project", bson.D{
			{"id", 0},
			{"payment_due", 1},
			{"table_number", "$_id.table_number"},
			{"order_items", 1},
		}},
	}

	result, err := orderItemCollection.Aggregate(c, mongo.Pipeline{
		matchStage,
		lookupStage,
		unwindStage,
		lookupOrderStage,
		unwindOrderStage,
		lookupTableStage,
		unwindTableStage,
		projectStage,
		groupStage,
		projectStage2,
	})

	if err != nil {
		panic(err)
	}

	if err = result.All(c, &OrderItems); err != nil {
		panic(err)
	}

	defer cancel()
	return OrderItems, err
}

func GetOrderItem() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		orderItemId := ctx.Param("order_item_id")
		var orderItem models.OrderItem

		err := orderItemCollection.FindOne(c, bson.M{"order_item_id": orderItemId}).Decode(&orderItem)
		defer cancel()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error occured while listing the item"})
			return
		}

		ctx.JSON(http.StatusOK, orderItem)
	}
}

func CreateOrderItem() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var OrderItemPack OrderItemPack
		var order models.Order

		if err := ctx.BindJSON(&OrderItemPack); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		order.Order_Date, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		orderItemsToBeInserted := []interface{}{}
		order.Table_id = &OrderItemPack.Table_id
		order_id := OrderItemOrderCreator(order)

		for _, orderItem := range OrderItemPack.Oder_items {
			orderItem.Order_id = &order_id

			validationErr := validate.Struct(orderItem)
			if validationErr != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": validationErr})
				return
			}

			orderItem.ID = primitive.NewObjectID()
			orderItem.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.Order_item_id = orderItem.ID.Hex()

			var num = toFixed(orderItem.Unit_price, 2)
			orderItem.Unit_price = num
			orderItemsToBeInserted = append(orderItemsToBeInserted, orderItem)
		}

		result, err := orderItemCollection.InsertMany(c, orderItemsToBeInserted)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer cancel()
		ctx.JSON(http.StatusOK, result)
	}
}

func UpdateOrderItem() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var orderItem models.OrderItem

		orderItemId := ctx.Param("order_item_id")
		filter := bson.M{"order_item_id": orderItemId}

		var updateObj primitive.D

		if orderItem.Unit_price != 0.0 {
			updateObj = append(updateObj, bson.E{"unit_price", orderItem.Unit_price})
		}

		if orderItem.Quantity != "" {
			updateObj = append(updateObj, bson.E{"quantity", orderItem.Quantity})
		}

		if orderItem.Food_id != nil {
			updateObj = append(updateObj, bson.E{"food_id", orderItem.Food_id})
		}

		orderItem.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", orderItem.Updated_at})

		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		result, err := orderCollection.UpdateOne(c, filter, bson.D{
			{"$set", updateObj},
		}, &opt)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Order item update failed"})
			return
		}

		defer cancel()
		ctx.JSON(http.StatusOK, result)
	}
}
