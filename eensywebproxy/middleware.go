package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	razorpay "github.com/razorpay/razorpay-go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// rzpPayments : will help get / post payment objects from/on eensymachines database
// when the payment is done= success/failure this will receive a post request
// gives a chance to the eensymachines database to update the payments
// this will only verify the payment and NOT update the order
// For the order to be updated, the client would have to send patch on the order
func rzpPayments(c *gin.Context) {
	if c.Request.Method == "POST" {
		// when the payment is successufully completed - we get a post request here denoting save in the database
		// this will also verifiy the signature of the payment so as to be verified
		defer c.Request.Body.Close()
		byt, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("invalid details for payment confirmation")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		paymntDone := RzpPaymentDone{}
		if err := json.Unmarshal(byt, &paymntDone); err != nil {
			log.WithFields(log.Fields{
				"err": err,
			}).Error("error unmarshaling order details")
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		log.WithFields(log.Fields{
			"payment": paymntDone.PaymntID,
			"order":   paymntDone.OrderID,
		}).Info("Payment confirmed, verified")
		// https://razorpay.com/docs/payments/server-integration/go/payment-gateway/build-integration#16-verify-payment-signature
		yes, apiErr := verifyRzpPayment(paymntDone)
		if apiErr != nil {
			Dispatch(apiErr, c, "rzpPayments/POST")
			return
		}
		if !yes {
			log.Warnf("payment %s not verified", paymntDone.PaymntID)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		log.Infof("payment %s for order %s verified", paymntDone.PaymntID, paymntDone.OrderID)
		c.AbortWithStatus(http.StatusOK)
		return
	}
}

// dbConnect : collection pointer injection onto the context
func dbConnect(client *mongo.Client, collName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		coll := client.Database("eensyweb").Collection(collName)
		c.Set("dbcoll", coll)
		log.Infof("now connected to database, coll: %s", collName)
	}
}

// rzpOrder : this deals with one order at a time
func rzpOrder(c *gin.Context) {
	orderId := c.Param("oid")
	client := razorpay.NewClient(os.Getenv("RZPKEY"), os.Getenv("RZPSECRET"))
	val, ok := c.Get("dbcoll")
	if !ok {
		Dispatch(&ApiErr{fmt.Errorf("failed to get DB connection"), ErrDbConn}, c, "rzpOrders/dbcoll")
		return
	}
	ordersColl := val.(*mongo.Collection)
	if c.Request.Method == "PATCH" {
		/*Once the payment is complete - success / failure the order needs to be patched for the details */
		// Getting order details from Rzp
		// This will have order details after the payment has been updated
		body, err := client.Order.Fetch(orderId, nil, nil)
		byt, _ := json.Marshal(body)
		rzpOrder := RzpOrder{}
		json.Unmarshal(byt, &rzpOrder)
		// Now getting the payments for the order
		body, err = client.Order.Payments(orderId, nil, nil)
		payments := struct {
			Items []PaymentDetails `json:"items"`
		}{}
		byt, _ = json.Marshal(body)
		json.Unmarshal(byt, &payments)
		// attaching the payments to the order object
		for _, p := range payments.Items {
			rzpOrder.Payments = append(rzpOrder.Payments, p)
		}
		// Updating the Eensy Machines database
		// Order object is replaced
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// We are replacing the object and not updating it
		_, err = ordersColl.ReplaceOne(ctx, bson.M{"id": rzpOrder.Id}, rzpOrder)
		if err != nil {
			Dispatch(&ApiErr{err, ErrQry}, c, "rzpOrders/Order.Create")
			return
		}
		c.AbortWithStatus(http.StatusOK)
		return
	}
}

// rzpOrders : middleware function that is called when RazorPay API is invoked
func rzpOrders(c *gin.Context) {
	// when the client would want to create a new order
	// Creating new Razory pay order
	// TODO: this client being created has to come from another middleware
	client := razorpay.NewClient(os.Getenv("RZPKEY"), os.Getenv("RZPSECRET"))
	defer c.Request.Body.Close()
	val, ok := c.Get("dbcoll")
	if !ok {
		Dispatch(&ApiErr{nil, ErrDbConn}, c, "rzpOrders/dbcoll")
		return
	}
	ordersColl := val.(*mongo.Collection)
	if c.Request.Method == "POST" {
		byt, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			Dispatch(&ApiErr{err, ErrInvlPayload}, c, "rzpOrders/ReadAll")
			return
		}
		payload := RzpOrderPayload{}
		if err := json.Unmarshal(byt, &payload); err != nil {
			Dispatch(&ApiErr{err, ErrInvlPayload}, c, "rzpOrders/Unmarshal")
			return
		}
		order, apiErr := CreateOrder(payload, ordersColl, client)
		if apiErr != nil {
			Dispatch(apiErr, c, "rzpOrders/POST")
		}
		// // generating a new uuid for the receiptm
		// // recep_uuid(last 12 characters)
		// uuid := uuid.New().String()
		// recipId := fmt.Sprintf("recep_%s", uuid[len(uuid)-12:])
		// data := map[string]interface{}{
		// 	"amount":          order.Amount,
		// 	"currency":        order.Currency,
		// 	"receipt":         recipId,
		// 	"partial_payment": order.PartialPay,
		// 	"notes":           order.Notes,
		// }
		// body, err := client.Order.Create(data, nil)
		// if err != nil {
		// 	Dispatch(&ApiErr{err, ErrExtApi}, c, "rzpOrders/Order.Create")
		// 	return
		// }
		// byt, _ = json.Marshal(body)
		// rzpOrder := RzpOrder{}
		// json.Unmarshal(byt, &rzpOrder)
		// // this newly created order needs to be pushed to the datbase
		// val, ok := c.Get("dbcoll")
		// if !ok {
		// 	Dispatch(&ApiErr{err, ErrDbConn}, c, "rzpOrders/dbcoll")
		// 	return
		// }
		// ordersColl := val.(*mongo.Collection)
		// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		// defer cancel()
		// _, err = ordersColl.InsertOne(ctx, rzpOrder)
		// if err != nil {
		// 	Dispatch(&ApiErr{err, ErrQry}, c, "rzpOrders/Order.Create")
		// 	return
		// }
		// log.WithFields(log.Fields{
		// 	"id": body["id"],
		// }).Info("Created new order")
		// When the order was created we send back the order for payment processing
		c.AbortWithStatusJSON(http.StatusOK, order)
		// ++++++++++++++++
	}
}
