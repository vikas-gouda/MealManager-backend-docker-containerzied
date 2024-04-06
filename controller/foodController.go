package controller

import (
	"context"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/vikas-gouda/go-restraunt-mangement/database"
	"github.com/vikas-gouda/go-restraunt-mangement/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")
var menuCollection *mongo.Collection = database.OpenCollection(database.Client, "menu")

var validate = validator.New()

func GetFoods() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		recordPerPagge, err := strconv.Atoi((ctx.Query("recordPerPage")))
		if err != nil || recordPerPagge < 1 {
			recordPerPagge = 10
		}

		page, err := strconv.Atoi(ctx.Query("page"))
		if err != nil || page < 1 {
			page = 1
		}

		startIndex := (page - 1) * recordPerPagge
		startIndex, err = strconv.Atoi(ctx.Query("startIndex"))

		matchStage := bson.D{
			{"$match", bson.D{{}}},
		}

		groupStage := bson.D{
			{"$group", bson.D{
				{"_id", bson.D{{"_id", "null"}}},
				{"total_count", bson.D{{"$sum", "1"}}},
				{"data", bson.D{{"$push", "$ROOT"}}},
			}},
		}

		projectStage := bson.D{
			{
				"$project", bson.D{
					{"_id", 0},
					{"total_count", 1},
					{"food_items", bson.D{{"$slice", []interface{}{"$data", startIndex, recordPerPagge}}}},
				},
			},
		}

		result, err := foodCollection.Aggregate(c, mongo.Pipeline{
			matchStage, groupStage, projectStage,
		})
		defer cancel()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing food items"})
			return
		}

		var allFoods []bson.M

		if err = result.All(c, &allFoods); err != nil {
			log.Fatal(err.Error())
			return
		}

		ctx.JSON(http.StatusOK, allFoods[0])

	}
}

func GetFood() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		foodId := ctx.Param("food_id")

		var food models.Food

		err := foodCollection.FindOne(c, bson.M{"food_id": foodId}).Decode(&food)
		defer cancel()

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error occured while the food id"})
			return
		}

		ctx.JSON(http.StatusOK, food)
	}
}

func CreateFood() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var menu models.Menu
		var food models.Food

		if err := ctx.BindJSON(&food); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validatonErr := validate.Struct(food)
		if validatonErr != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": validatonErr.Error()})
			return
		}

		err := menuCollection.FindOne(c, bson.M{"menu_id": food.Menu_id}).Decode(&menu)
		defer cancel()

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "menu was not found"})
		}

		food.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		food.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		food.ID = primitive.NewObjectID()
		food.Food_id = food.ID.Hex()

		food.Price = toFixed(*&food.Price, 2)

		result, insertErr := foodCollection.InsertOne(c, food)
		if insertErr != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Food was not created"})
			return
		}

		defer cancel()
		ctx.JSON(http.StatusOK, result)

	}
}

func UpdateFood() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var food models.Food
		var menu models.Menu

		if err := ctx.BindJSON(&food); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		foodId := ctx.Param("food_id")
		menuId := ctx.Param("menu_id")

		err := menuCollection.FindOne(c, bson.M{"menu_id": menuId}).Decode(&menu)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Menu doesnt exist"})
			return
		}

		var updateObj primitive.D

		if food.Name != "" {
			updateObj = append(updateObj, bson.E{"name", food.Name})
		}

		if food.Price != 0.0 {
			updateObj = append(updateObj, bson.E{"food_price", food.Price})
		}

		if food.Food_image != "" {
			updateObj = append(updateObj, bson.E{"food_image", food.Food_image})
		}

		if food.Menu_id != nil {
			updateObj = append(updateObj, bson.E{"menu_id", food.Menu_id})
		}

		food.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", food.Updated_at})

		upsert := true
		filter := bson.M{"food_id": foodId}

		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		result, err := foodCollection.UpdateOne(c, filter, bson.D{
			{"$set", updateObj},
		}, &opt)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the food"})
			return
		}

		defer cancel()
		ctx.JSON(http.StatusOK, result)
	}
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
