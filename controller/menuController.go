package controller

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vikas-gouda/go-restraunt-mangement/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetMenus() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		result, err := menuCollection.Find(context.TODO(), bson.M{})
		defer cancel()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error occured while listing the menu items"})
			return
		}

		var allMenus []bson.M
		if err = result.All(c, &allMenus); err != nil {
			log.Fatal(err)
			return
		}

		ctx.JSON(http.StatusOK, allMenus)
	}
}

func GetMenu() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var menu models.Menu
		var menuID = ctx.Param("menu_id")

		err := menuCollection.FindOne(c, bson.M{"menu_id": menuID}).Decode(&menu)
		defer cancel()

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching the menu"})
			return
		}

		ctx.JSON(http.StatusOK, menu)

	}
}

func CreateMenu() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var menu models.Menu
		if err := ctx.BindJSON(&menu); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validateErr := validate.Struct(menu)
		if validateErr != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": validateErr.Error()})
			return
		}

		menu.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		menu.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		menu.ID = primitive.NewObjectID()
		menu.Menu_id = menu.ID.Hex()

		result, err := menuCollection.InsertOne(c, menu)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while creating the menu"})
			return
		}

		defer cancel()

		ctx.JSON(http.StatusOK, result)

	}
}

func UpdateMenu() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var menu models.Menu

		if err := ctx.BindJSON(&menu); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		menuID := ctx.Param("menu_id")
		filter := bson.M{"menu_id": menuID}

		var updateObj primitive.D

		if (menu.Start_date != time.Time{} && menu.End_date != time.Time{}) {
			if !inTimeSpan(*&menu.Start_date, *&menu.End_date, time.Now()) {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Kindly retype the time"})
				defer cancel()
				return
			}

			updateObj = append(updateObj, bson.E{"start_date", menu.Start_date})
			updateObj = append(updateObj, bson.E{"end_date", menu.End_date})

			if menu.Name != "" {
				updateObj = append(updateObj, bson.E{"name", menu.Name})
			}
			if menu.Category != "" {
				updateObj = append(updateObj, bson.E{"category", menu.Category})
			}

			menu.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

			updateObj = append(updateObj, bson.E{"updated_at", menu.Updated_at})

			upsert := true

			opt := options.UpdateOptions{
				Upsert: &upsert,
			}

			result, err := menuCollection.UpdateOne(c, filter, bson.D{
				{"$set", updateObj},
			}, &opt)

			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Menu Update Failed"})
			}

			defer cancel()
			ctx.JSON(http.StatusOK, result)
		}
	}
}

func inTimeSpan(start, end, check time.Time) bool {
	return start.After(time.Now()) && end.After(time.Now())
}
