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
	"github.com/google/uuid"
	razorpay "github.com/razorpay/razorpay-go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

// RzpOrderPayload : when creating an order this is the shape of the data incoming in the request as payload
type RzpOrderPayload struct {
	Amount     int    `json:"amount"`
	PartialPay bool   `json:"partial_payment"`
	Currency   string `json:"currency"`
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

// rzpPayments : will help get / post payment objects from/on eensymachines database
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
		// TODO: here the signature needs to be verified before we can call it a valid transaction
		c.AbortWithStatus(http.StatusOK)
		return
	}
}

/* ApiErrCode : error code on which the error dispatch hinges
 */
type ApiErrCode int

// Enum for types of error
const (
	ErrInvlPayload ApiErrCode = iota + 8400
	ErrExtApi
	ErrDbConn
	ErrQry
)

// IApiErr : interface over which a mixing function will Dispatch will consume error
type IApiErr interface {
	Err() string
	Code() int
	ErrMsg() string
}

// ApiErr : implements the interface, and the objct that middleware will sedn for disptach
type ApiErr struct {
	e    error
	code ApiErrCode
}

//================= IApiErr implementation=========
func (ape *ApiErr) Err() string {
	return ape.e.Error()
}
func (ape *ApiErr) Code() int {
	switch ape.code {
	case ErrInvlPayload:
		return http.StatusBadRequest
	case ErrExtApi, ErrDbConn:
		return http.StatusBadGateway
	case ErrQry:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
func (ape *ApiErr) ErrMsg() string {
	switch ape.code {
	case ErrInvlPayload:
		return "Request inputs invalid, try checking the inputs and resend"
	case ErrExtApi:
		return "Failed external api call. This happens when a 3rd party server has error."
	case ErrDbConn:
		return "Failed database connection. Try in sometime."
	case ErrQry:
		return "failed to get data, one or more operations on the server failed."
	default:
		return "Unknown error! Something on the server seems to be broken"
	}
}

// ======================
// Dispatch : Mixing functionthat will evntually cross fit the error to the gin Context and also log the error
// Uses the IApiErr interface

func Dispatch(apie IApiErr, c *gin.Context, trace string) {
	log.WithFields(log.Fields{
		"err": apie.Err(),
	}).Errorf("%s:%s", c.Request.URL.Path, trace)
	c.AbortWithStatusJSON(apie.Code(), gin.H{
		"err": apie.ErrMsg(),
		// this the error message that is displayed in the front end
	})
	return
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
			"notes":           map[string]interface{}{},
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
