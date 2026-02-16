package ton

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

// CreateWallet creates a wallet from seed phrase and returns its address.
func (t *Ton) CreateWallet(seedPhrase string) (string, error) {
	w, err := wallet.FromSeedWithOptions(t.api, strings.Split(seedPhrase, " "), wallet.ConfigV5R1Final{
		NetworkGlobalID: wallet.MainnetGlobalID,
	})
	if err != nil {
		return "", errors.Wrap(err, "FromSeed err:")

	}

	return w.WalletAddress().String(), nil
}
