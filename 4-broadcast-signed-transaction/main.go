package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
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
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	// Parse broadcasted transaction response
	var response struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		panic(err)
	}
	fmt.Printf("Broadcasted transaction hash: https://"+network+".etherscan.io/tx/%s\n", response.ID)

	// * Get the number of confirmations for the transaction https://docs.blockdaemon.com/reference/gettxconfirmations
	fmt.Printf("Sleeping for 60s before checking confirmations...\n")
	time.Sleep(80 * time.Second)
	res, err = http.Get(url + "/universal/v1/" + protocol + "/" + network + "/tx/" + response.ID + "/confirmations?apiKey=" + apiKey)
	resbodyBytes, _ := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	fmt.Println(string(resbodyBytes))

}
