package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vikas-gouda/go-restraunt-mangement/database"
	"github.com/vikas-gouda/go-restraunt-mangement/helpers"
	"github.com/vikas-gouda/go-restraunt-mangement/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")

func GetUsers() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		recordPerPage, err := strconv.Atoi(ctx.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		page, err1 := strconv.Atoi(ctx.Query("page"))
		if err1 != nil || page < 1 {
			page = 1
		}

		startIndex := (page - 1) * recordPerPage
		startIndex, err = strconv.Atoi(ctx.Query("startIndex"))

		matchStage := bson.D{{"$match", bson.D{{}}}}
		projectStage := bson.D{
			{
				"$project", bson.D{
					{"_id", 0},
					{"total_count", 1},
					{"user_items", bson.D{
						{"$slice", []interface{}{"$data", startIndex, recordPerPage}},
					}},
				},
			},
		}

		result, err := userCollection.Aggregate(c, mongo.Pipeline{
			matchStage, projectStage,
		})

		defer cancel()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "error ocured while listing user items"})
			return
		}

		var allUsers []bson.M
		if err = result.All(c, &allUsers); err != nil {
			log.Fatal(err.Error())
		}

		ctx.JSON(http.StatusOK, allUsers[0])

	}
}

func GetUser() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		userId := ctx.Param("user_id")

		var user models.User

		err := userCollection.FindOne(c, bson.M{"user_id": userId}).Decode(&user)

		defer cancel()

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while listing the user"})
		}

		ctx.JSON(http.StatusOK, user)
	}
}

func SignUp() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var user models.User

		if err := ctx.BindJSON(&user); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validateErr := validate.Struct(user)
		if validateErr != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": validateErr.Error()})
			return
		}

		countEmail, err := userCollection.CountDocuments(c, bson.M{"email": user.Email})
		defer cancel()

		if err != nil {
			log.Panic(err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "while checking for the email"})
			return
		}

		password := HashPassword(user.Password)
		user.Password = password

		countPhone, err := userCollection.CountDocuments(c, bson.M{"phone": user.Phone})
		if err != nil {
			log.Panic(err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "while checking for the phone number"})
			return
		}

		if countEmail > 0 || countPhone > 0 {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "The email or phone number already exists"})
			return
		}

		user.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()

		token, refreshToken, err := helpers.GenerateAllTokens(user.Email, user.First_name, user.Last_name, user.User_id)

		user.Token = token
		user.Refresh_token = refreshToken

		result, insertErr := userCollection.InsertOne(c, user)

		if insertErr != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User item is not created"})
			return
		}

		defer cancel()

		ctx.JSON(http.StatusOK, result)
	}
}

func Login() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var user models.User
		var foundUser models.User

		if err := ctx.BindJSON(&user); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := userCollection.FindOne(c, bson.M{"email": user.Email}).Decode(&user)

		defer cancel()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
			return
		}

		passwordisValid, msg := VerifyPassword(user.Password, *&foundUser.Password)
		defer cancel()
		if passwordisValid != true {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		token, refreshToken, _ := helpers.GenerateAllTokens(foundUser.Email, foundUser.First_name, foundUser.Last_name, foundUser.User_id)

		helpers.UpdateAllTokens(token, refreshToken, foundUser.User_id)

		ctx.JSON(http.StatusOK, foundUser)

	}
}

func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}

	return string(bytes)
}

func VerifyPassword(userPassword string, providedPassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := true
	msg := ""

	if err != nil {
		msg = fmt.Sprintf("Email or password is incorrect")
		check = false
	}

	return check, msg
}
