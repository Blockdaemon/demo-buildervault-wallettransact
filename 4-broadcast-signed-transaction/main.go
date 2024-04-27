package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
)

var protocol = "ethereum"
var network = "sepolia"
var url = "https://svc.blockdaemon.com/tx/v1/" + protocol + "-" + network + "/"

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
	chainID := big.NewInt(11155111) // ToDo get chain ID from unsignedTX or onchain

	// ! Combine the unsigned transaction and the signature to create a signed transaction
	//signer := types.NewEIP155Signer(chainID)	// Legacy signer pre eip1559
	//signedTx, err := unsignedTx.WithSignature(signer, transactionHashSignatureBytes)
	signedTx, err := unsignedTx.WithSignature(types.NewLondonSigner(chainID), transactionHashSignatureBytes)
	if err != nil {
		panic(err)
	}

	// ! Serialize the signed transaction for broadcast
	raw, err := signedTx.MarshalBinary()
	if err != nil {
		panic(err)
	}
	fmt.Printf("\nRaw signed transaction (RLP encoded): %x", raw)

	// ! Broadcast the signed transaction to the blockchain
	requestBody := struct {
		ID string `json:"tx"`
	}{
		ID: hex.EncodeToString(raw),
	}

	reqBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println(err)
		return
	}

	req, _ := http.NewRequest("POST", url+"send", bytes.NewReader(reqBodyBytes))
	req.Header.Add("X-API-Key", apiKey)

	res, _ := http.DefaultClient.Do(req)
	resbodyBytes, _ := io.ReadAll(res.Body)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatal("HTTP request ", res.StatusCode, " error:", http.StatusText(res.StatusCode), "n/", string(resbodyBytes))
	} else {
		fmt.Printf("\nBroadcasted transaction hash: https://"+network+".etherscan.io/tx/%s\n", signedTx.Hash().String())
	}

}
