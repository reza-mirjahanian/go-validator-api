package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"net/http"
	"net/url"
)

func startRestAPIs() {
	cfg, err := loadConfig(".env")
	checkIfErr(err)

	// Parse RPC endpoint URL
	rpcEndpointURL, err := url.Parse(cfg.RPC_ENDPOINT)
	checkIfErr(err)

	// Init Web3 client
	client := BuildWeb3(rpcEndpointURL, rate.Limit(cfg.RPC_RATE_LIMITER_NUMBER))

	// Init Gin ginRouter
	// https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
	ginRouter := gin.Default()
	ginRouter.ForwardedByClientIP = true
	err = ginRouter.SetTrustedProxies(cfg.GIN_TRUSTED_PROXIES_LIST)
	checkIfErr(err)

	ginRouter.GET("/syncduties/:slot", syncDutiesHandler(client))
	ginRouter.GET("/blockreward/:slot", blockRewardHandler(client))

	err = ginRouter.Run(cfg.GIN_SERVER_ADDRESS)
	checkIfErr(err)
}

// blockRewardHandler handles requests for block rewards
func blockRewardHandler(client *Web3) gin.HandlerFunc {
	return func(context *gin.Context) {
		slot := context.Param("slot")
		reward, status, err := client.GetBlockRewardAndStatusBySlot(context, slot)
		if err != nil {
			handleError(context, err)
			return
		}
		context.IndentedJSON(http.StatusOK, gin.H{"reward": reward, "status": status})
	}
}

// syncDutiesHandler handles requests for sync duties
func syncDutiesHandler(client *Web3) gin.HandlerFunc {
	return func(c *gin.Context) {
		slot := c.Param("slot")
		pubKeysList, err := client.GetSyncCommitteeDuties(slot)
		if err != nil {
			handleError(c, err)
			return
		}
		c.IndentedJSON(http.StatusOK, pubKeysList)
	}
}

// handleError handles different error types and sets the appropriate HTTP status codes
func handleError(context *gin.Context, err error) {
	var slotUnavailableError *SlotUnavailableError
	var slotTooFarInFutureError *SlotTooFarInFutureError
	switch {
	case errors.As(err, &slotUnavailableError):
		context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.As(err, &slotTooFarInFutureError):
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		context.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
