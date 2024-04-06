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

var tableCollection *mongo.Collection = database.OpenCollection(database.Client, "table")

func GetTables() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		result, err := tableCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var allTables []bson.M
		if err := result.All(c, &allTables); err != nil {
			log.Fatal(err)
			return
		}

		ctx.JSON(http.StatusOK, allTables)
	}
}

func GetTable() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		table_id := ctx.Param("table_id")

		var table models.Table

		err := tableCollection.FindOne(c, bson.M{"table_id": table_id}).Decode(&table)
		defer cancel()

		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, table)
	}
}

func CreateTable() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var table models.Table

		if err := ctx.BindJSON(&table); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(table)
		if validationErr != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": validationErr.Error()})
			return
		}

		table.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		table.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		table.ID = primitive.NewObjectID()
		table.Table_id = table.ID.Hex()

		result, insertErr := tableCollection.InsertOne(c, table)
		if insertErr != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while inserting"})
			return
		}
		defer cancel()

		ctx.JSON(http.StatusOK, result)

	}
}

func UpdateTable() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var table models.Table

		tableId := ctx.Param("table_id")

		if err := ctx.BindJSON(&table); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var updateObj primitive.D

		if table.Number_of_guests != 0 {
			updateObj = append(updateObj, bson.E{"number_of_guests", table.Number_of_guests})
		}

		if table.Table_number != 0 {
			updateObj = append(updateObj, bson.E{"table_number", table.Table_number})
		}

		table.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		filter := bson.M{"table_id": tableId}

		result, err := tableCollection.UpdateOne(c, filter, bson.D{{"$set", updateObj}}, &opt)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Table item update failed"})
			return
		}

		defer cancel()
		ctx.JSON(http.StatusOK, result)

	}
}
