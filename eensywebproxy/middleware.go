package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	razorpay "github.com/razorpay/razorpay-go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// RzpOrderPayload : when creating an order this is the shape of the data incoming in the request as payload
type RzpOrderPayload struct {
	Amount     int                    `json:"amount"`
	PartialPay bool                   `json:"partial_payment"`
	Currency   string                 `json:"currency"`
	Notes      map[string]interface{} `json:"notes"`
}

// RzpOrder : outgoing payload when the order is created
// I thought of using this but then sending out a simple map[string]interface{} is sufficing
// XXX: can used in the future
type RzpOrder struct {
	Amount     int    `json:"amount"`
	AmountDue  int    `json:"amount_due"`
	AmountPaid int    `json:"amount_paid"`
	Attempts   int    `json:"attempts"`
	Currency   string `json:"currency"`
	Id         string `json:"id"`
	Receipt    string `json:"receipt"`
	Status     string `json:"status"`
}
type RzpPaymentDone struct {
	PaymntID string `json:"razorpay_payment_id"`
	OrderID  string `json:"razorpay_order_id"`
	Signtr   string `json:"razorpay_signature"`
}

// verifyRzpPayment: verifies the razorpay payment from the signature
// creates a new SHA signature with order id and payment id using the same crypto algorithm
// then compares the hash with payment signature
// Error incase there is crypto failure or bad inputs
func verifyRzpPayment(done RzpPaymentDone) (bool, IApiErr) {
	secret := os.Getenv("RZPSECRET")
	if secret == "" {
		log.Error("razorpay secret is not loaded on environment")
		return false, &ApiErr{fmt.Errorf("verifyRzpPayment:invalid razorpay secret key, check environment if loaded"), ErrEnv}
	}
	data := fmt.Sprintf("%s|%s", done.OrderID, done.PaymntID)
	h := hmac.New(sha256.New, []byte(secret))
	_, err := h.Write([]byte(data))
	if err != nil {
		log.WithFields(log.Fields{
			"order":   done.OrderID,
			"payment": done.PaymntID,
			"err":     err,
		}).Error("failed to create sha signature")
		return false, &ApiErr{fmt.Errorf("verifyRzpPayment:failed to create sha256 signature for verification %s", err), ErrEncrypt}
	}
	sha := hex.EncodeToString(h.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(sha), []byte(done.Signtr)) == 1 {
		return true, nil
	}
	log.WithFields(log.Fields{
		"order":   done.OrderID,
		"payment": done.PaymntID,
	}).Warn("payment signature is not authenticated")
	return false, nil
}

// rzpPayments : will help get / post payment objects from/on eensymachines database
func rzpPayments(c *gin.Context) {
	if c.Request.Method == "POST" {
		val, ok := c.Get("dbcoll")
		if !ok {
			Dispatch(&ApiErr{fmt.Errorf("failed to get dbconnection in middleware"), ErrDbConn}, c, "rzpPayments/dbcoll")
			return
		}
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
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		ordersColl := val.(*mongo.Collection)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		filter := bson.M{"id": paymntDone.OrderID}
		// TODO: for now we are pushing only the payment id
		// but we need to push more payment details than this
		// these details need to come from client side
		// RzpPaymentDone the object needs to change
		update := bson.M{"$addToSet": bson.M{"payments": paymntDone.PaymntID}}
		_, err = ordersColl.UpdateOne(ctx, filter, update)
		if err != nil {
			Dispatch(&ApiErr{e: fmt.Errorf("failed to update orders of verified payments"), code: ErrQry}, c, "rzpPayments/UpdateOne")
			return
		}
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

// rzpOrders : middleware function that is called when RazorPay API is invoked
func rzpOrders(c *gin.Context) {
	// when the client would want to create a new order
	// Creating new Razory pay order
	// TODO: this client being created has to come from another middleware
	client := razorpay.NewClient(os.Getenv("RZPKEY"), os.Getenv("RZPSECRET"))
	defer c.Request.Body.Close()
	if c.Request.Method == "POST" {
		byt, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			Dispatch(&ApiErr{err, ErrInvlPayload}, c, "rzpOrders/ReadAll")
			return
		}
		order := RzpOrderPayload{}
		if err := json.Unmarshal(byt, &order); err != nil {
			Dispatch(&ApiErr{err, ErrInvlPayload}, c, "rzpOrders/Unmarshal")
			return
		}
		// generating a new uuid for the receiptm
		// recep_uuid(last 12 characters)
		uuid := uuid.New().String()
		recipId := fmt.Sprintf("recep_%s", uuid[len(uuid)-12:])
		data := map[string]interface{}{
			"amount":          order.Amount,
			"currency":        order.Currency,
			"receipt":         recipId,
			"partial_payment": order.PartialPay,
			"notes":           order.Notes,
		}
		body, err := client.Order.Create(data, nil)
		if err != nil {
			Dispatch(&ApiErr{err, ErrExtApi}, c, "rzpOrders/Order.Create")
			return
		}
		// this newly created order needs to be pushed to the datbase
		val, ok := c.Get("dbcoll")
		if !ok {
			Dispatch(&ApiErr{err, ErrDbConn}, c, "rzpOrders/dbcoll")
			return
		}
		ordersColl := val.(*mongo.Collection)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = ordersColl.InsertOne(ctx, body)
		if err != nil {
			Dispatch(&ApiErr{err, ErrQry}, c, "rzpOrders/Order.Create")
			return
		}
		log.WithFields(log.Fields{
			"id": body["id"],
		}).Info("Created new order")
		// When the order was created we send back the order for payment processing
		c.AbortWithStatusJSON(http.StatusOK, body)
		// ++++++++++++++++
	}
}
