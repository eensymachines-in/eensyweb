package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	razorpay "github.com/razorpay/razorpay-go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

// dbConnect : collection pointer injection onto the context
// also will inject the rzp client in the context
func dbConnect(client *mongo.Client, collName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		coll := client.Database("eensyweb").Collection(collName)
		c.Set("dbcoll", coll)
		c.Set("rzpcl", razorpay.NewClient(os.Getenv("RZPKEY"), os.Getenv("RZPSECRET")))
	}
}

// rzpPayments : will help get / post payment objects from/on eensymachines database
// when the payment is done= success/failure this will receive a post request
// this will record the payment in the database, and also patch the order details onto the database.
func rzpPayments(c *gin.Context) {
	val, ok := c.Get("dbcoll")
	if !ok {
		Dispatch(&ApiErr{nil, ErrDbConn}, c, "rzpOrders/dbcoll")
		return
	}
	ordersColl := val.(*mongo.Collection)
	val, ok = c.Get("rzpcl")
	if !ok {
		Dispatch(&ApiErr{nil, ErrExtApi}, c, "rzpOrders/rzpcl")
		return
	}
	rzpcl := val.(*razorpay.Client)
	if c.Request.Method == "POST" {
		// when the payment is successufully completed - we get a post request here denoting save in the database
		// this will also verifiy the signature of the payment so as to be verified
		defer c.Request.Body.Close()
		byt, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			Dispatch(&ApiErr{e: err, code: ErrInvlPayload}, c, "rzpPayments/POST")
			return
		}
		paymntDone := RzpPaymentDone{}
		if err := json.Unmarshal(byt, &paymntDone); err != nil {
			Dispatch(&ApiErr{e: err, code: ErrInvlPayload}, c, "rzpPayments/POST")
			return
		}
		// https://razorpay.com/docs/payments/server-integration/go/payment-gateway/build-integration#16-verify-payment-signature
		if paymntDone.Signtr != "" {
			// Incase the payment fails signature sent in would be emtpty string
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
			log.WithFields(log.Fields{
				"payment": paymntDone.PaymntID,
				"order":   paymntDone.OrderID,
			}).Info("Payment confirmed, verified")
		}
		// Now patching the order details from the rzp server
		if err := UpdateOrder(paymntDone.OrderID, ordersColl, rzpcl); err != nil {
			Dispatch(err, c, "rzpPayments/POST")
		}
		log.Infof("updated order for payment %s", paymntDone.PaymntID)
		// partial view that can show the payment confirmation
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
		c.AbortWithStatusJSON(http.StatusOK, order)
		// ++++++++++++++++
	}
}

// // rzpOrder : this deals with one order at a time
// func rzpOrder(c *gin.Context) {
// 	orderId := c.Param("oid")
// 	client := razorpay.NewClient(os.Getenv("RZPKEY"), os.Getenv("RZPSECRET"))
// 	val, ok := c.Get("dbcoll")
// 	if !ok {
// 		Dispatch(&ApiErr{fmt.Errorf("failed to get DB connection"), ErrDbConn}, c, "rzpOrders/dbcoll")
// 		return
// 	}
// 	ordersColl := val.(*mongo.Collection)
// 	if c.Request.Method == "PATCH" {
// 		/*Once the payment is complete - success / failure the order needs to be patched for the details */
// 		// Getting order details from Rzp
// 		// This will have order details after the payment has been updated
// 		if err := UpdateOrder(orderId, ordersColl, client); err != nil {
// 			Dispatch(err, c, "rzpOrder/PATCH")
// 		}
// 		c.AbortWithStatus(http.StatusOK)
// 		return
// 	}
// }
