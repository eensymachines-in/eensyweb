package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

/* ApiErrCode : error code on which the error dispatch hinges
 */
type ApiErrCode int

// Enum for types of error
const (
	ErrInvlPayload ApiErrCode = iota + 8400
	ErrExtApi
	ErrDbConn
	ErrQry
	ErrEnv
	ErrEncrypt
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
// Err : this is used to log the message on the server stdout
// as much details as required here since this is logged on the server
func (ape *ApiErr) Err() string {
	return ape.e.Error()
}

// Code: this is used to convert to http response code
func (ape *ApiErr) Code() int {
	switch ape.code {
	case ErrInvlPayload:
		return http.StatusBadRequest
	case ErrExtApi, ErrDbConn:
		return http.StatusBadGateway
	case ErrQry, ErrEnv, ErrEncrypt:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// ErrMsg : this is the message that the client will see and display to the user
// hide as much details as you can here about the server working
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
	case ErrEnv:
		return "server settings were invalid, wait for the admin to fix it and try again"
	case ErrEncrypt:
		return "one or more encryption operations on the server has failed."
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
