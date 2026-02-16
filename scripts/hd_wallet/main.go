package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/xssnick/tonutils-go/ton/wallet"
)

func main() {
	mnemonic := flag.String("mnemonic", "", "24-word mnemonic phrase (space-separated)")
	subwalletID := flag.Uint("subwallet-id", 0, "Subwallet ID for address derivation (different IDs = different addresses)")
	flag.Parse()

	if *mnemonic == "" {
		log.Fatal("mnemonic is required")
	}

	words := strings.Split(*mnemonic, " ")
	if len(words) != 24 {
		log.Fatalf("expected 24 words, got %d", len(words))
	}

	// Convert TON mnemonic to private key
	privateKey, err := wallet.SeedToPrivateKeyWithOptions(words)
	if err != nil {
		log.Fatalf("failed to derive private key: %v", err)
	}

	// Extract public key
	publicKey := privateKey.Public().(ed25519.PublicKey)

	// Get address using V4R2 with custom subwallet ID
	// Different subwallet IDs produce different contract addresses
	// All addresses are controlled by the same private key
	swID := uint32(*subwalletID)
	addr, err := wallet.AddressFromPubKey(publicKey, wallet.V4R2, swID)
	if err != nil {
		log.Fatalf("failed to get address: %v", err)
	}

	// Format addresses
	bounceableStr := addr.String() // Default is bounceable (EQ...)

	// Get non-bounceable format (UQ...)
	addr.SetBounce(false)
	nonBounceableStr := addr.String()

	// Output
	fmt.Println("========================================")
	fmt.Println("TON Wallet (Subwallet Derivation)")
	fmt.Println("========================================")
	fmt.Printf("Subwallet ID:          %d\n", swID)
	fmt.Println("----------------------------------------")
	fmt.Printf("Private Key:           %s\n", hex.EncodeToString(privateKey.Seed()))
	fmt.Printf("Public Key:            %s\n", hex.EncodeToString(publicKey))
	fmt.Println("----------------------------------------")
	fmt.Printf("Address (Bounceable):     %s\n", bounceableStr)
	fmt.Printf("Address (Non-Bounceable): %s\n", nonBounceableStr)
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Note: All subwallets share the same private key.")
	fmt.Println("      Different subwallet IDs = different contract addresses.")
}
