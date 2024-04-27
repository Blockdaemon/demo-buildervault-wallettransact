package main

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"gitlab.com/Blockdaemon/go-tsm-sdkv2/tsm"
)

var serverMtlsPublicKeys = map[int]string{
	0: "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEaWLFxRxgLQHJ662gcd2LfPFYKDmI\n8AlzFUu/MFR0Pb5d0JYSBL/HAUR5/1OXfEV18riJZJCeOa1gxNocwzqZ9Q==\n-----END PUBLIC KEY-----\n",
	1: "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAErPzIZwRgiFpBgIDYCzfRxEgvasus\nHa4qlwWnJ0TnlGgjcfD5Bp40J9HnOdlBkzhtVWq5PiLEMaFWdApTkRBT9Q==\n-----END PUBLIC KEY-----\n",
	2: "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEyaLwUY4A99EDvqGMjBT2Q/M3zydm\nOniFOZicnwdvnJTMgXw8LAqLee+0VFIUZbxRPTvN1c1ORoD8+2xJ0VPglg==\n-----END PUBLIC KEY-----\n",
}

func main() {

	// ! Specify the Builder Vailt TSM master key, created in step 1, to sign via the derived wallet chain path m/44/51/0/0
	masterKeyID := "..."

	// ! Specify the NewLondonSigner unsigned transaction hash created in step 2
	unsignedTxHash := "..."

	// Decode server public keys to bytes for use in TLS client authentication
	serverPKIXPublicKeys := make([][]byte, len(serverMtlsPublicKeys))
	for i := range serverMtlsPublicKeys {
		block, rest := pem.Decode([]byte(serverMtlsPublicKeys[i]))
		if block == nil || len(rest) != 0 {
			panic("error decoding server public key (no block data)")
		}
		serverPublicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			panic(err)
		}
		serverPKIXPublicKeys[i], err = x509.MarshalPKIXPublicKey(serverPublicKey)
		if err != nil {
			panic(err)
		}
	}

	// Create TSM SDK clients with mTLS authentication and public key pinning
	clients := make([]*tsm.Client, len(serverMtlsPublicKeys))
	for i := range clients {
		config, err := tsm.Configuration{URL: fmt.Sprintf("https://tsm-sandbox.prd.wallet.blockdaemon.app:%v", 8080+i)}.WithMTLSAuthentication("./client.key", "./client.crt", serverPKIXPublicKeys[i])
		if err != nil {
			panic(err)
		}
		clients[i], err = tsm.NewClient(config)
		if err != nil {
			panic(err)
		}
	}

	// ToDo: see why static node indices are not sufficient and why dynamic public keys are needed
	// The public keys of the other players to encrypt MPC protocol data end-to-end
	playerB64Pubkeys := []string{
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEtDFBfanInAMHNKKDG2RW/DiSnYeI7scVvfHIwUIRdbPH0gBrsilqxlvsKZTakN8om/Psc6igO+224X8T0J9eMg==",
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEqvSkhonTeNhlETse8v3X7g4p100EW9xIqg4aRpD8yDXgB0UYjhd+gFtOCsRT2lRhuqNForqqC+YnBsJeZ4ANxg==",
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEBaHCIiViexaVaPuER4tE6oJE3IBA0U//GlB51C1kXkT07liVc51uWuYk78wi4e1unxC95QbeIfnDCG2i43fW3g==",
	}

	playerPubkeys := map[int][]byte{}
	playerIds := []int{0, 1, 2}
	// iterate over other players public keys and convert them
	for i := range playerIds {
		pubkey, err := base64.StdEncoding.DecodeString(playerB64Pubkeys[i])
		if err != nil {
			panic(err)
		}
		playerPubkeys[playerIds[i]] = pubkey
	}

	chainPath := []uint32{44, 60, 0, 0}
	//! convert hex string to bytes
	unsignedTxHashBytes, _ := hex.DecodeString(unsignedTxHash)
	partialSignaturesLock := sync.Mutex{}
	partialSignatures := make([][]byte, 0)
	//sessionConfig := tsm.NewStaticSessionConfig(tsm.GenerateSessionID(),3)
	Players := []int{0, 1, 2}
	sessionConfig := tsm.NewSessionConfig(tsm.GenerateSessionID(), Players, playerPubkeys)
	ctx := context.Background()
	var eg errgroup.Group
	for _, client := range clients {
		client := client
		eg.Go(func() error {
			partialSignResult, err := client.ECDSA().Sign(ctx, sessionConfig, masterKeyID, chainPath, unsignedTxHashBytes)
			if err != nil {
				return err
			}
			partialSignaturesLock.Lock()
			partialSignatures = append(partialSignatures, partialSignResult.PartialSignature)
			partialSignaturesLock.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		panic(err)
	}

	signature, err := tsm.ECDSAFinalizeSignature(unsignedTxHashBytes, partialSignatures)
	if err != nil {
		panic(err)
	}

	// Construct signature in R|S|V format
	sigBytes := make([]byte, 2*32+1)
	copy(sigBytes[0:32], signature.R())
	copy(sigBytes[32:64], signature.S())
	sigBytes[64] = byte(signature.RecoveryID())
	fmt.Println("TX hash signature:", hex.EncodeToString(sigBytes))
}
