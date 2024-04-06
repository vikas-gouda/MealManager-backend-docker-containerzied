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

type InvoiceViewFormat struct {
	Invoice_id       string
	Payment_method   string
	Order_id         string
	Payment_status   *string
	Payment_due      interface{}
	Table_number     interface{}
	Payment_due_date time.Time
	Order_details    interface{}
}

var InvoiceCollection *mongo.Collection = database.OpenCollection(database.Client, "invoice")

func GetInvoices() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		result, err := InvoiceCollection.Find(context.Background(), bson.M{})
		defer cancel()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var allInvoices []bson.M
		if err := result.All(c, &allInvoices); err != nil {
			log.Fatal(err)
			return
		}

		ctx.JSON(http.StatusOK, allInvoices)
	}
}

func GetInvoice() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		invoiceId := ctx.Param("invoice_id")

		var invoice models.Invoice

		err := InvoiceCollection.FindOne(c, bson.M{"invoice_id": invoiceId}).Decode(&invoice)
		defer cancel()

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while listing invoice item"})
		}

		var invoiceView InvoiceViewFormat

		allOrderItems, err := ItemsByOrder(*invoice.Order_id)
		invoiceView.Order_id = *invoice.Order_id
		invoiceView.Payment_due_date = invoice.Payment_due_date

		invoiceView.Payment_method = "null"
		if invoice.Payment_method != "" {
			invoiceView.Payment_method = invoice.Payment_method
		}

		invoiceView.Invoice_id = invoice.Invoice_id
		invoiceView.Payment_status = &invoice.Payment_status
		invoiceView.Payment_due = allOrderItems[0]["payment_due"]
		invoiceView.Table_number = allOrderItems[0]["table_number"]
		invoiceView.Order_details = allOrderItems[0]["order_items"]

		ctx.JSON(http.StatusOK, invoiceView)
	}
}

func CreateInvoice() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var invoice models.Invoice
		var order models.Invoice

		if err := ctx.BindJSON(&invoice); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(invoice)
		if validationErr != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": validationErr.Error()})
			return
		}

		err := orderCollection.FindOne(c, bson.M{"oder_id": invoice.Order_id}).Decode(&order)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "order not found"})
			return
		}

		invoice.Payment_due_date, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		invoice.ID = primitive.NewObjectID()
		invoice.Invoice_id = invoice.ID.Hex()

		result, insertErr := InvoiceCollection.InsertOne(c, order)
		if insertErr != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error while inserting"})
			return
		}
		defer cancel()

		ctx.JSON(http.StatusOK, result)
	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var c, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var invoice models.Invoice
		var order models.Order

		if err := ctx.BindJSON(&invoice); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		invoiceId := ctx.Param("invoice_id")
		orderId := ctx.Param("order_id")

		err := orderCollection.FindOne(c, bson.M{"order_id": orderId}).Decode(&order)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "order not found"})
			return
		}

		var updateObj primitive.D

		if invoice.Payment_method != "" {
			updateObj = append(updateObj, bson.E{"payment_method", invoice.Payment_method})
		}

		if invoice.Payment_status == "" {
			invoice.Payment_status = "PENDING"
			updateObj = append(updateObj, bson.E{"payment_status", invoice.Payment_status})
		}

		if (invoice.Payment_due_date != time.Time{}) {
			updateObj = append(updateObj, bson.E{"payment_due_date", invoice.Payment_due_date})
		}

		invoice.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", invoice.Updated_at})

		upsert := true
		filter := bson.M{"invoice_id": invoiceId}
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		result, err := InvoiceCollection.UpdateOne(c, filter, bson.D{
			{"$set", updateObj},
		}, &opt)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the Invoice"})
			return
		}

		defer cancel()
		ctx.JSON(http.StatusOK, result)

	}
}
