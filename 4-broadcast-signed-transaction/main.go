package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
)

var protocol = "ethereum"
var network = "sepolia"
var url = "https://svc.blockdaemon.com"

func main() {
	// ! Set the wallet public keu created in step 1
	walletPublicKey := "..."

	// ! Set the rawUnsignedTransaction created in step 2
	rawUnsignedTransaction := "..."

	// ! Set the DER transactionHashSignature created in step 3
	transactionHashSignature := "..."

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

	// * Compile the unsigned tx with the signatureand broadcast to the blockchain https://docs.blockdaemon.com/reference/txcompileandsend-txapi
	txRequest := struct {
		Unsigned_tx string `json:"unsigned_tx"`
		Signature   string `json:"signature"`
		Public_key  string `json:"public_key"`
	}{
		Unsigned_tx: rawUnsignedTransaction,
		Signature:   transactionHashSignature,
		Public_key:  walletPublicKey,
	}

	reqBody, err := json.Marshal(txRequest)
	if err != nil {
		panic(err)
	}

	res, err := http.Post(url+"/tx/v1/"+protocol+"-"+network+"/compile_and_send?apiKey="+apiKey, "application/json", bytes.NewReader(reqBody))
	resbodyBytes, _ := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	// Check HTTP status code
	if res.StatusCode != http.StatusOK {
		panic(fmt.Errorf("invalid status code:%d %s %s", res.StatusCode, http.StatusText(res.StatusCode), resbodyBytes))
	}

	// Parse broadcasted transaction response
	var response struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		panic(err)
	}
	fmt.Printf("Broadcasted transaction hash: https://"+network+".etherscan.io/tx/%s\n", response.ID)

}
