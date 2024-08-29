package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
)

var protocol = "ethereum"
var network = "sepolia"
var url = "https://svc.blockdaemon.com"
var address = "..."                                                   // ! Set the wallet address created in step 1
var destinationAddress = "0x52b09e2c73849B25F9b0328e2d4b444e9bd1EF30" // Optional - override destination address which defaults back to faucet
var amount = "0.001"                                                  // Set the amount to send in ETH

func main() {

	// Access token is required
	apiKey := os.Getenv("ACCESS_TOKEN")
	if apiKey == "" {
		panic(fmt.Errorf("env variable 'ACCESS_TOKEN' must be set"))
	}

	// * Get account balance https://docs.blockdaemon.com/reference/getlistofbalancesbyaddress
	res, err := http.Get(url + "/universal/v1/" + protocol + "/" + network + "/account/" + address + "?apiKey=" + apiKey)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	// Parse account balance response
	var account []struct {
		Currency struct {
			Symbol   string `json:"symbol"`
			Decimals int    `json:"decimals"`
		} `json:"currency"`
		ConfirmedBalance string `json:"confirmed_balance"`
	}
	if err := json.NewDecoder(res.Body).Decode(&account); err != nil {
		panic(err)
	}

	// Print account balance in float
	balanceInt, _ := new(big.Int).SetString(account[0].ConfirmedBalance, 10)
	balanceFloat := new(big.Float).SetInt(balanceInt)
	balanceFloat.Mul(balanceFloat, big.NewFloat(math.Pow10(-account[0].Currency.Decimals)))
	fmt.Printf("Balance at account %s: %s %s\n", address, balanceFloat.Text('f', 18), account[0].Currency.Symbol)

	// * Transaction request - MaxFeePerGas and MaxPriorityFeePerGas are estimated automatically https://docs.blockdaemon.com/reference/txcreate-txapi
	txRequest := struct {
		From  string `json:"from"`
		To    string `json:"to"`
		Value string `json:"value"`
	}{
		From:  address,
		To:    destinationAddress,
		Value: amount,
	}

	reqBody, err := json.Marshal(txRequest)
	if err != nil {
		panic(err)
	}

	// Post unsigned transaction request
	res, err = http.Post(url+"/tx/v1/"+protocol+"-"+network+"/create?apiKey="+apiKey, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	// Check HTTP status code
	if res.StatusCode != http.StatusOK {
		panic(fmt.Errorf("invalid status code:%d %s", res.StatusCode, http.StatusText(res.StatusCode)))
	}

	// Parse unsigned transaction response
	var response struct {
		UnsignedTx string `json:"unsigned_tx"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		panic(err)
	}
	fmt.Printf("Raw unsigned tx: %s\n", response.UnsignedTx)

	// Deserialize the rawUnsignedTransaction
	unsignedTx := &types.Transaction{}
	unsignedTxBytes, _ := hex.DecodeString(response.UnsignedTx)
	if err := unsignedTx.UnmarshalBinary(unsignedTxBytes); err != nil {
		panic(err)
	}

	// * create a NewLondonSigner EIP1559 transaction type hash
	signer := types.NewLondonSigner(unsignedTx.ChainId())
	fmt.Printf("Raw unsigned NewLondonSigner tx hash: %s\n", hex.EncodeToString(signer.Hash(unsignedTx).Bytes()))

}
