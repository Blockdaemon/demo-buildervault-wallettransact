package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
)

var protocol = "ethereum"
var network = "sepolia"
var chainID = big.NewInt(11155111)
var url = "https://svc.blockdaemon.com/universal/v1/" + protocol + "/" + network + "/"

func main() {
	// ! Set the transactionHashSignature created in step 4
	transactionHashSignature := "..."

	// ! Set the rawUnsignedTransaction created in step 3
	rawUnsignedTransaction := "..."

	// Access token is required
	apiKey := os.Getenv("ACCESS_TOKEN")
	if apiKey == "" {
		panic(fmt.Errorf("env variable 'ACCESS_TOKEN' must be set"))
	}

	// Deserialize the rawUnsignedTransaction
	unsignedTx := &types.Transaction{}
	unsignedTxBytes, _ := hex.DecodeString(rawUnsignedTransaction)
	err := unsignedTx.UnmarshalBinary(unsignedTxBytes)
	if err != nil {
		panic(err)
	}

	// Deserialize the transactionHashSignature
	transactionHashSignatureBytes, _ := hex.DecodeString(transactionHashSignature)

	// * Combine the unsigned transaction and the signature to create a signed transaction
	signedTx, err := unsignedTx.WithSignature(types.NewLondonSigner(chainID), transactionHashSignatureBytes)
	if err != nil {
		panic(err)
	}

	// * Serialize the signed transaction for broadcast
	raw, err := signedTx.MarshalBinary()
	if err != nil {
		panic(err)
	}
	fmt.Printf("\nRaw signed tx (RLP encoded): %x\n", raw)

	// * Broadcast the signed transaction to the blockchain https://docs.blockdaemon.com/reference/txsend
	sendRequest := struct {
		TX string `json:"tx"`
	}{
		TX: hex.EncodeToString(raw),
	}

	reqBody, err := json.Marshal(sendRequest)
	if err != nil {
		panic(err)
	}

	res, _ := http.Post(url+"tx/send?apiKey="+apiKey, "application/json", bytes.NewReader(reqBody))
	resbodyBytes, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		panic(fmt.Errorf("HTTP request %d error: %sn/ %s", res.StatusCode, http.StatusText(res.StatusCode), resbodyBytes))
	} else {
		fmt.Printf("\nBroadcasted transaction hash: https://"+network+".etherscan.io/tx/%s\n", signedTx.Hash().String())
	}

}
