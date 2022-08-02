package main

import (
	"testing"

	razorpay "github.com/razorpay/razorpay-go"
)

func TestGetRzpOrders(t *testing.T) {
	client := razorpay.NewClient("rzp_test_Z4AumzgwmBpgQv", "TQXlGbKAXteB8UYhzWqgrB2A")
	body, err := client.Order.Fetch("order_K0fK2m02H1athc", nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(body)
}
