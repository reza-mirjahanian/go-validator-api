package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/time/rate"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
)

type beaconBlockDetailResponse struct {
	Data struct {
		Message struct {
			Body struct {
				Eth1data struct {
					BlockHash string `json:"block_hash"`
				} `json:"eth1_data"`
			} `json:"body"`
		} `json:"message"`
	} `json:"data"`
}

type syncCommitteesResponse struct {
	Data struct {
		Validators []string `json:"validators"`
	} `json:"data"`
}

type validatorsDetailResponse struct {
	Data []struct {
		Validator struct {
			Pubkey string `json:"pubkey"`
		} `json:"validator"`
	} `json:"data"`
}

type BeaconHeader struct {
	Data []struct {
		Header struct {
			Message struct {
				Slot string `json:"slot"`
			} `json:"message"`
		} `json:"header"`
	} `json:"data"`
}

type rateLimitTransport struct {
	rateLimiter *rate.Limiter
	transport   http.RoundTripper
}

// RoundTrip is an implementation of the RoundTripper interface that enforces the rate limit.
func (r *rateLimitTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx := context.Background()
	if waitErr := r.rateLimiter.Wait(ctx); waitErr != nil {
		return nil, waitErr
	}
	return r.transport.RoundTrip(request)
}

type SlotUnavailableError struct {
	Message string `json:"message"`
}

func (e *SlotUnavailableError) Error() string {
	return e.Message
}

type SlotTooFarInFutureError struct {
	Message string `json:"message"`
}

func (e *SlotTooFarInFutureError) Error() string {
	return e.Message
}

type Web3 struct {
	EthClient  *ethclient.Client
	HTTPClient *http.Client
	endpoint   *url.URL
}

func BuildWeb3(endpoint *url.URL, requestsPerSecond rate.Limit) *Web3 {
	rateLimiter := rate.NewLimiter(requestsPerSecond, 1)

	clientWithRateLimit := &http.Client{
		Transport: &rateLimitTransport{
			rateLimiter: rateLimiter,
			transport:   http.DefaultTransport,
		},
	}

	connection, err := rpc.DialOptions(context.Background(), endpoint.String(), rpc.WithHTTPClient(clientWithRateLimit))
	if err != nil {
		log.Printf("Error in InitializeWeb3 -> rpc.DialOptions: %s", err.Error())
		return nil
	}

	return &Web3{
		endpoint:   endpoint,
		HTTPClient: clientWithRateLimit,
		EthClient:  ethclient.NewClient(connection),
	}
}

func (w *Web3) getEndpointData(endpoint string, operation string, result interface{}) error {
	// Create the request
	httpRequest, reqErr := http.NewRequest("GET", endpoint, nil)
	if reqErr != nil {
		log.Printf("Failed to create request for: %s with error: %s", endpoint, reqErr.Error())
		return reqErr
	}

	log.Println("Requesting GET data from: ", endpoint)

	httpRequest.Header.Set("Accept", "application/json")

	// Send the request
	httpResponse, respErr := w.HTTPClient.Do(httpRequest)
	if respErr != nil {
		log.Printf("Failed to execute operation: %s with error: %s", operation, respErr.Error())
		return respErr
	}
	defer func(body io.ReadCloser) {
		closeErr := body.Close()
		if closeErr != nil {
			log.Printf("Error closing response body for operation: %s with error: %s", operation, closeErr.Error())
		}
	}(httpResponse.Body)

	// Read the response body
	responseBody, readErr := io.ReadAll(httpResponse.Body)
	if readErr != nil {
		log.Printf("Failed to read response body for operation: %s with error: %s", operation, readErr.Error())
		return readErr
	}

	// Handle different status codes
	switch httpResponse.StatusCode {
	case http.StatusNotFound:
		return &SlotTooFarInFutureError{Message: "Requested slot is too far in the future"}
	case http.StatusBadRequest:
		return &SlotUnavailableError{Message: "Slot does not exist"}
	}

	// Unmarshal the response body into the provided result interface
	if unmarshalErr := json.Unmarshal(responseBody, result); unmarshalErr != nil {
		return unmarshalErr
	}

	return nil
}

// fetchBlockHash retrieves the block hash for a given slot from the Beacon chain.
func (w *Web3) fetchBlockHash(slot string) (common.Hash, error) {
	// Construct the API endpoint URL for the Beacon chain block detail.
	// https://docs.blastapi.io/blast-documentation/apis-documentation/core-api/ethereum/holesky-beacon/beacon/eth-v2-beacon-blocks-block_id
	endpoint := w.endpoint.String() + "/eth/v2/beacon/blocks/" + slot

	// Define a variable to store the Beacon chain block detail response.
	var blockInfo beaconBlockDetailResponse

	// Fetch the Beacon chain block detail from the API endpoint.
	err := w.getEndpointData(endpoint, "beacon block info", &blockInfo)
	if err != nil {
		// If there's an error, return an empty hash and the error.
		return common.Hash{}, err
	}

	// Extract the block hash from the Beacon chain block detail response.
	hash := blockInfo.Data.Message.Body.Eth1data.BlockHash

	// Convert the block hash from a hexadecimal string to a byte slice.
	return common.HexToHash(hash), nil
}

// Fetch the current slot  from the beacon chain.
func (w *Web3) getCurrentHeadSlot() *big.Int {
	// Build the API endpoint URL for the beacon chain headers.
	endpoint := w.endpoint.String() + "/eth/v1/beacon/headers"

	// Define the response struct for the API request.
	var beaconHeader BeaconHeader

	// Fetch the data from the API endpoint and store it in the response struct.
	err := w.getEndpointData(endpoint, "current slot ", &beaconHeader)
	if err != nil {
		return big.NewInt(0)
	}

	// Parse the slot string from the response struct into a big.Int.
	slot, valid := new(big.Int).SetString(beaconHeader.Data[0].Header.Message.Slot, 10)
	if !valid {
		// If the slot string is not a valid big.Int, return a big.Int with value 0.
		return big.NewInt(0)
	}

	return slot
}

func (w *Web3) GetBlockRewardAndStatusBySlot(ctx context.Context, slotStr string) (*string, *string, error) {
	err := w.validateSlot(slotStr)
	if err != nil {
		return nil, nil, err
	}

	blockHash, hashErr := w.fetchBlockHash(slotStr)
	if hashErr != nil {
		return nil, nil, hashErr
	}

	block, blockErr := w.EthClient.BlockByHash(ctx, blockHash)
	if blockErr != nil {
		log.Printf("Error in EthClient.BlockByHash: %s", blockErr.Error())
		return nil, nil, blockErr
	}

	baseFee := block.BaseFee()
	totalBurntFees := new(big.Int).Mul(baseFee, big.NewInt(int64(block.GasUsed())))
	totalTxCosts := new(big.Int).SetInt64(0)
	status := "vanilla"

	for _, tx := range block.Transactions() {
		receipt, receiptErr := w.EthClient.TransactionReceipt(ctx, tx.Hash())
		txCost := tx.Cost()
		gasPrice := tx.GasPrice()

		if receiptErr == nil {
			txCost = new(big.Int).Mul(receipt.EffectiveGasPrice, big.NewInt(int64(receipt.GasUsed)))
			gasPrice = receipt.EffectiveGasPrice
		}

		// Determine if the transaction is a MEV transaction
		if gasPrice.Cmp(new(big.Int).Mul(baseFee, big.NewInt(3))) == 1 {
			status = "mev"
		}

		totalTxCosts = new(big.Int).Add(totalTxCosts, txCost)
	}

	reward := new(big.Int).Sub(totalTxCosts, totalBurntFees)
	rewardFloat := new(big.Float).Quo(new(big.Float).SetInt(reward), new(big.Float).SetInt(big.NewInt(1000000000)))
	rewardStr := rewardFloat.Text('f', 9)

	return &rewardStr, &status, nil
}

func (w *Web3) validateSlot(slotStr string) error {
	slotInt, valid := new(big.Int).SetString(slotStr, 10)
	if !valid {
		return errors.New("cannot convert slot to big.Int")
	}

	// Check if the slot is before the Paris merge slot
	if slotInt.Cmp(big.NewInt(4700012)) != 1 {
		return &SlotUnavailableError{Message: "Slot is missing"}
	}

	currentSlot := w.getCurrentHeadSlot()
	if slotInt.Cmp(currentSlot) == 1 {
		return &SlotTooFarInFutureError{Message: "Slot is in the future"}
	}
	return nil
}

func (w *Web3) GetSyncCommitteeDuties(slotStr string) ([]string, error) {
	err := w.validateSlot(slotStr)
	if err != nil {
		return nil, err
	}

	// Fetch validator indexes for the given slot
	validators, fetchErr := w.fetchSyncCommitteesValidatorIndexes(slotStr)
	if fetchErr != nil {
		return nil, fetchErr
	}

	// Get public keys of sync committee members
	syncCommitteeKeys, keysErr := w.getPubKeysOfSyncCommittees(slotStr, validators)
	if keysErr != nil {
		return nil, keysErr
	}

	return syncCommitteeKeys, nil
}

// fetchSyncCommitteesValidatorIndexes retrieves the validator indexes for the sync committees of a given slot .
func (w *Web3) fetchSyncCommitteesValidatorIndexes(slot string) ([]string, error) {
	// Construct the API endpoint URL for the sync committees of the given slot.
	//https://docs.blastapi.io/blast-documentation/apis-documentation/core-api/ethereum/holesky-beacon/beacon/eth-v1-beacon-states-state_id-sync_committees
	endpoint := w.endpoint.String() + "/eth/v1/beacon/states/" + slot + "/sync_committees"

	// Define a variable to store the sync committees response.
	var response syncCommitteesResponse

	// Fetch the sync committees data from the API endpoint.
	err := w.getEndpointData(endpoint, "sync committees", &response)
	if err != nil {
		return nil, err
	}

	// Extract the validator indexes from the sync committees response.
	validatorIndexes := response.Data.Validators

	// Return the validator indexes.
	return validatorIndexes, nil
}

// RetrievePubKeysOfSyncCommittees retrievves the public keys of the validators in the sync committees for a given slot.
func (w *Web3) getPubKeysOfSyncCommittees(slot string, validatorIndices []string) ([]string, error) {
	// Build the API endpoint URL with validator indices as query parameters.
	// https://docs.blastapi.io/blast-documentation/apis-documentation/core-api/ethereum/holesky-beacon/beacon/eth-v1-beacon-states-state_id-validators
	endpoint := w.endpoint.String() + "/eth/v1/beacon/states/" + slot + "/validators"
	for index, validatorIndex := range validatorIndices {
		if index == 0 {
			endpoint += "?"
		}
		endpoint += "id=" + validatorIndex
		if index != len(validatorIndices)-1 {
			endpoint += "&"
		}
	}

	// Define the detailResponse struct for the API request.
	var detailResponse validatorsDetailResponse

	// Fetch the data from the API endpoint and store it in the detailResponse struct.
	err := w.getEndpointData(endpoint, "receive pubkeys of validators", &detailResponse)
	if err != nil {
		return nil, err
	}

	// Extract the public keys from the detailResponse struct and store them in a slice.
	var pubKeys []string
	for _, info := range detailResponse.Data {
		pubKeys = append(pubKeys, info.Validator.Pubkey)
	}

	return pubKeys, nil
}
