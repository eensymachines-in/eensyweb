package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/razorpay/razorpay-go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

/*This file shall house all the objects and their allied functions relating to razorpay payments
 */

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
	// Amounts are in paise and not Rupees hence the integer space required is large
	Amount     int64 `json:"amount"`
	AmountDue  int64 `json:"amount_due"`
	AmountPaid int64 `json:"amount_paid"`
	// attempts cannot be more than 100 in anycase hence a shorter version of the integer
	Attempts int8   `json:"attempts"`
	Currency string `json:"currency"`
	Id       string `json:"id"`
	Receipt  string `json:"receipt"`
	Status   string `json:"status"`
	// TODO: Payments attempts are in slice
	Payments []PaymentDetails `json:"payments,omitempty"`
}
type RzpPaymentDone struct {
	PaymntID string `json:"razorpay_payment_id"`
	OrderID  string `json:"razorpay_order_id"`
	Signtr   string `json:"razorpay_signature"`
}

// PaymentDetails : details of a single payment
type PaymentDetails struct {
	Id           string `json:"id"`
	InvoiceId    string `json:"invoice_id"`
	Amount       int64  `json:"amount"`
	Refunded     int64  `json:"amount_refunded"`
	Fee          int64  `json:"fee"`
	Tax          int64  `json:"tax"`
	Captured     bool   `json:"captured"`
	Intrntl      bool   `json:"international"`
	RefundStatus string `json:"refund_status"`
	Status       string `json:"status"`
	Bank         string `json:"bank"`
	Method       string `json:"method"`
	Contact      string `json:"contact"`
	CreatedAt    int64  `json:"created_at"`
	ErrCode      int    `json:"error_code"`
	ErrDesc      string `json:"error_description"`
	ErrReason    string `json:"error_reason"`
	ErrSrc       string `json:"error_source"`
	ErrStep      string `json:"error_step"`
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

// Principal functions we write here work on the object models above and work on Rzp account and the database below
// CreateOrder : will create an order from the payload details, will then send it across to razorpay, and also create a copy onto eensymachines database
// This order object is without any payment details
// pl		: this is the payload we receive in the request, contains basic order details
// coll		: pointer to mongo collection with active database connection
// rzpcl	: razorpay client
func CreateOrder(pl RzpOrderPayload, coll *mongo.Collection, rzpcl *razorpay.Client) (*RzpOrder, IApiErr) {
	uuid := uuid.New().String()
	recipId := fmt.Sprintf("recep_%s", uuid[len(uuid)-12:])
	data := map[string]interface{}{
		"amount":          pl.Amount,
		"currency":        pl.Currency,
		"receipt":         recipId, //this is generated
		"partial_payment": pl.PartialPay,
		"notes":           pl.Notes,
	}
	body, err := rzpcl.Order.Create(data, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"amount":  pl.Amount,
			"receipt": recipId,
		}).Error("failed to create order with razorpay")
		return nil, &ApiErr{e: err, code: ErrExtApi}
	}
	byt, err := json.Marshal(body)
	if err != nil {
		log.WithFields(log.Fields{
			"amount":  pl.Amount,
			"receipt": recipId,
		}).Error("failed marshal razorpay order details")
		return nil, &ApiErr{e: err, code: ErrEncrypt}
	}
	order := RzpOrder{}
	if err := json.Unmarshal(byt, &order); err != nil {
		log.WithFields(log.Fields{
			"amount":  pl.Amount,
			"receipt": recipId,
		}).Error("failed unmarshal razorpay order details")
		return nil, &ApiErr{e: err, code: ErrEncrypt}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = coll.InsertOne(ctx, order)
	if err != nil {
		log.WithFields(log.Fields{
			"amount":  pl.Amount,
			"receipt": recipId,
		}).Error("failed insert order details to database")
		return nil, &ApiErr{e: err, code: ErrQry}
	}
	log.WithFields(log.Fields{
		"order_id": order.Id,
	}).Info("created new order")
	return &order, nil
}

// UpdateOrder after the payment has been done this helps to synchronize various fields on the order including the payments fields
// Fields on order : Amount paid, due, attempts of payments, payments
// Once the payment is done, RZP services will have the latest data for the order - this needs to be synchronized here on the Eensymachines db
func UpdateOrder(orderId string, coll *mongo.Collection, rzpcl *razorpay.Client) IApiErr {
	if orderId == "" || coll == nil || rzpcl == nil {
		return &ApiErr{e: fmt.Errorf("orderid, dbcollection or the rzp client is invalid"), code: ErrDbConn}
	}
	body, err := rzpcl.Order.Fetch(orderId, nil, nil)
	if err != nil {
		return &ApiErr{e: fmt.Errorf("failed to fetch order details %s: %s", orderId, err), code: ErrExtApi}
	}
	byt, _ := json.Marshal(body)
	rzpOrder := RzpOrder{}
	json.Unmarshal(byt, &rzpOrder)
	// Now getting the payments for the order
	body, err = rzpcl.Order.Payments(orderId, nil, nil)
	if err != nil {
		return &ApiErr{e: fmt.Errorf("failed to fetch payment details for order %s: %s", orderId, err), code: ErrExtApi}
	}
	payments := struct {
		Items []PaymentDetails `json:"items"`
	}{}
	byt, _ = json.Marshal(body)
	json.Unmarshal(byt, &payments)
	// attaching the payments to the order object
	for _, p := range payments.Items {
		rzpOrder.Payments = append(rzpOrder.Payments, p)
	}
	// Updating our database with payments done
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// We are replacing the object and not updating it
	_, err = coll.ReplaceOne(ctx, bson.M{"id": rzpOrder.Id}, rzpOrder)
	if err != nil {
		return &ApiErr{e: fmt.Errorf("failed to replace order in database %s: %s", orderId, err), code: ErrQry}
	}
	return nil
}
