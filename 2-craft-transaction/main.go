package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
)

var protocol = "ethereum"
var network = "sepolia"
var url = "https://svc.blockdaemon.com/universal/v1/" + protocol + "/" + network + "/"
var address = "..."            // ! Set the wallet address created in step 1
var destinationAddress = "..." // ! Optional - override destination address which defaults back to faucet

type Account []struct {
	Currency struct {
		AssetPath string `json:"asset_path"`
		Symbol    string `json:"symbol"`
		Name      string `json:"name"`
		Decimals  int    `json:"decimals"`
		Type      string `json:"type"`
	} `json:"currency"`
	ConfirmedBalance string `json:"confirmed_balance"`
	PendingBalance   string `json:"pending_balance"`
	ConfirmedNonce   int    `json:"confirmed_nonce"`
	ConfirmedBlock   int    `json:"confirmed_block"`
}

type Transaction struct {
	Protocol struct {
		Ethereum struct {
			MaxFeePerGas         int64 `json:"maxFeePerGas"`
			MaxPriorityFeePerGas int64 `json:"maxPriorityFeePerGas"`
		} `json:"ethereum"`
	} `json:"protocol"`
	From string `json:"from"`
	To   []struct {
		Destination string `json:"destination"`
		Amount      string `json:"amount"`
	} `json:"to"`
}

func getBalance(address, apiKey string) string {
	req, _ := http.NewRequest("GET", url+"account/"+address, nil)
	req.Header.Add("X-API-Key", apiKey)

	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()

	var account Account
	if err := json.NewDecoder(res.Body).Decode(&account); err != nil {
		log.Fatal(err)
	}

	return account[0].ConfirmedBalance
}

func main() {

	// Access token is required
	apiKey := os.Getenv("ACCESS_TOKEN")
	if apiKey == "" {
		panic(fmt.Errorf("env variable 'ACCESS_TOKEN' must be set"))
	}

	confirmedBalance := getBalance(address, apiKey)
	fmt.Println("Balance at account:", address, "=", (confirmedBalance), "wei")

	// Post unsigned transaction request to send 0.003ETH to destinationAddress
	request := Transaction{
		To: []struct {
			Destination string `json:"destination"`
			Amount      string `json:"amount"`
		}{
			{
				Destination: destinationAddress,
				Amount:      "0.003",
			},
		},
		From: address,
		Protocol: struct {
			Ethereum struct {
				MaxFeePerGas         int64 "json:\"maxFeePerGas\""
				MaxPriorityFeePerGas int64 "json:\"maxPriorityFeePerGas\""
			} `json:"ethereum"`
		}{
			Ethereum: struct {
				MaxFeePerGas         int64 "json:\"maxFeePerGas\""
				MaxPriorityFeePerGas int64 "json:\"maxPriorityFeePerGas\""
			}{
				MaxFeePerGas:         200,
				MaxPriorityFeePerGas: 100,
			},
		},
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", url+"tx/create", bytes.NewReader(reqBody))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("X-API-Key", apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Fatalf("HTTP request %d %s", res.StatusCode, http.StatusText(res.StatusCode))
	}

	var response struct {
		UnsignedTx string `json:"unsigned_tx"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		log.Fatal(err)
	}

	// Deserialize the rawUnsignedTransaction
	unsignedTx := &types.Transaction{}
	unsignedTxBytes, _ := hex.DecodeString(response.UnsignedTx)
	if err := unsignedTx.UnmarshalBinary(unsignedTxBytes); err != nil {
		panic(err)
	}

	// create a NewLondonSigner for EIP 1559 transactions
	chainID := big.NewInt(11155111)
	signer := types.NewLondonSigner(chainID)
	fmt.Printf("Raw unsigned tx hash with NewLondonSigner: %s\n", hex.EncodeToString(signer.Hash(unsignedTx).Bytes()))

}
