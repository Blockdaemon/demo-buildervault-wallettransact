package main

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/Blockdaemon/go-tsm-sdkv2/tsm"
	"gitlab.com/Blockdaemon/go-tsm-sdkv2/tsm/tsmutils"
	"golang.org/x/sync/errgroup"
)

var serverMtlsPublicKeys = map[int]string{
	0: "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEaWLFxRxgLQHJ662gcd2LfPFYKDmI\n8AlzFUu/MFR0Pb5d0JYSBL/HAUR5/1OXfEV18riJZJCeOa1gxNocwzqZ9Q==\n-----END PUBLIC KEY-----\n",
	1: "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAErPzIZwRgiFpBgIDYCzfRxEgvasus\nHa4qlwWnJ0TnlGgjcfD5Bp40J9HnOdlBkzhtVWq5PiLEMaFWdApTkRBT9Q==\n-----END PUBLIC KEY-----\n",
	2: "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEyaLwUY4A99EDvqGMjBT2Q/M3zydm\nOniFOZicnwdvnJTMgXw8LAqLee+0VFIUZbxRPTvN1c1ORoD8+2xJ0VPglg==\n-----END PUBLIC KEY-----\n",
}

func main() {

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

	// The public keys of the other players to encrypt MPC protocol data end-to-end
	playerB64Pubkeys := []string{
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEtDFBfanInAMHNKKDG2RW/DiSnYeI7scVvfHIwUIRdbPH0gBrsilqxlvsKZTakN8om/Psc6igO+224X8T0J9eMg==",
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEqvSkhonTeNhlETse8v3X7g4p100EW9xIqg4aRpD8yDXgB0UYjhd+gFtOCsRT2lRhuqNForqqC+YnBsJeZ4ANxg==",
		"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEBaHCIiViexaVaPuER4tE6oJE3IBA0U//GlB51C1kXkT07liVc51uWuYk78wi4e1unxC95QbeIfnDCG2i43fW3g==",
	}

	// * Generate an ECDSA master key https://builder-vault-tsm.docs.blockdaemon.com/docs/getting-started-demo-tsm-golang
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

	threshold := 1 // The security threshold for this key
	keyGenPlayers := []int{0, 1, 2}
	sessionConfig := tsm.NewSessionConfig(tsm.GenerateSessionID(), keyGenPlayers, playerPubkeys)

	ctx := context.Background()
	masterKeyIDs := make([]string, len(clients))
	var eg errgroup.Group
	for i, client := range clients {
		client, i := client, i
		eg.Go(func() error {
			var err error
			masterKeyIDs[i], err = client.ECDSA().GenerateKey(ctx, sessionConfig, threshold, "secp256k1", "")
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		panic(err)
	}

	// Validate key IDs
	for i := 1; i < len(masterKeyIDs); i++ {
		if masterKeyIDs[0] != masterKeyIDs[i] {
			panic("key IDs do not match")
		}
	}
	masterKeyID := masterKeyIDs[0]
	fmt.Println("Builder Vault master key ID:", masterKeyID)

	// * Get the public key for the Ethereum derived key chain path m/44/60/0/0
	chainPath := []uint32{44, 60, 0, 0}
	pkixPublicKeys := make([][]byte, len(clients))
	for i, client := range clients {
		var err error
		pkixPublicKeys[i], err = client.ECDSA().PublicKey(ctx, masterKeyID, chainPath)
		if err != nil {
			panic(err)
		}
	}

	publicKeyBytes, err := tsmutils.PKIXPublicKeyToCompressedPoint(pkixPublicKeys[0])
	if err != nil {
		panic(err)
	}
	fmt.Println("Ethereum chain path m/44/60/0/0 derived wallet compressed public key:", hex.EncodeToString(publicKeyBytes))

	// Convert the public key into an Ethereum address
	publicKeyBytes, err = tsmutils.PKIXPublicKeyToUncompressedPoint(pkixPublicKeys[0])
	if err != nil {
		panic(err)
	}

	ecdsaPub, err := crypto.UnmarshalPubkey(publicKeyBytes)
	if err != nil {
		panic(err)
	}

	address := crypto.PubkeyToAddress(*ecdsaPub)
	fmt.Println("Ethereum chain path m/44/60/0/0 derived wallet address:", address)

}
