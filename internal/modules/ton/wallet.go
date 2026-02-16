package ton

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

// WalletInterface defines methods for wallet operations.
type WalletInterface interface {
	// GetAddress returns the wallet address (bounceable and non-bounceable) for a given subwallet ID.
	GetAddress(subwalletID uint32) (string, string, error)
	// SendTON sends TON from the master wallet to the given address. Amount in TON (e.g. 0.01).
	SendTON(ctx context.Context, toAddress string, amountTON float64, comment string, subwalletID uint32) (txHash []byte, err error)
}

// masterPrivateKey derives the same private key from master mnemonic for all subwallets.
// GetAddress and SendTON use this so the same subwalletID always corresponds to the same key and address.
func (t *Ton) masterPrivateKey() (ed25519.PrivateKey, error) {
	if t.cfg.Ton.MasterMnemonic == "" {
		return nil, errors.New("TON_MASTER_MNEMONIC is not set")
	}
	words := strings.Split(t.cfg.Ton.MasterMnemonic, " ")
	return wallet.SeedToPrivateKeyWithOptions(words)
}

// GetAddress generates a wallet address offline using the master mnemonic and subwallet ID.
func (t *Ton) GetAddress(subwalletID uint32) (string, string, error) {
	privateKey, err := t.masterPrivateKey()
	if err != nil {
		return "", "", err
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)

	addr, err := wallet.AddressFromPubKey(publicKey, wallet.V4R2, subwalletID)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get address from pubkey")
	}

	bounceableStr := addr.String()
	addr.SetBounce(false)
	nonBounceableStr := addr.String()

	return bounceableStr, nonBounceableStr, nil
}

// SendTON sends TON from the master wallet to the given address.
// toAddress: destination address (bounceable EQ... or non-bounceable UQ...).
// amountTON: amount in TON (e.g. 0.01).
// comment: optional text comment in the transfer (can be empty).
// subwalletID: subwallet of the master wallet to send from (same as in GetAddress).
// Returns transaction hash on success. Uses bounce from destination address (safe for wallets).
func (t *Ton) SendTON(ctx context.Context, toAddress string, amountTON float64, comment string, subwalletID uint32) ([]byte, error) {
	privateKey, err := t.masterPrivateKey()
	if err != nil {
		return nil, err
	}

	w, err := wallet.FromPrivateKeyWithOptions(privateKey, wallet.V4R2, wallet.WithAPI(t.api))
	if err != nil {
		return nil, errors.Wrap(err, "init wallet")
	}
	if subwalletID != w.GetSubwalletID() {
		w, err = w.GetSubwallet(subwalletID)
		if err != nil {
			return nil, errors.Wrap(err, "get subwallet")
		}
	}

	toAddr, err := address.ParseAddr(toAddress)
	if err != nil {
		return nil, errors.Wrap(err, "parse destination address")
	}

	amount := tlb.MustFromTON(fmt.Sprintf("%.9f", amountTON))
	msg, err := w.BuildTransfer(toAddr, amount, toAddr.IsBounceable(), comment)
	if err != nil {
		return nil, errors.Wrap(err, "build transfer")
	}

	txHash, err := w.SendManyWaitTxHash(ctx, []*wallet.Message{msg})
	if err != nil {
		return nil, errors.Wrap(err, "send transfer")
	}
	return txHash, nil
}
