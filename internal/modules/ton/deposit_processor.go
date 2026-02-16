package ton

import (
	"time"
)

const depositProcessorInterval = 5 * time.Second

// depositProcessorLoop runs every 5s: fetches transactions with status PROCESSING,
// finds customers by wallet_address == address_nonbounceable, credits ton_balance and sets DONE.
func (t *Ton) depositProcessorLoop() {
	defer t.wg.Done()

	ticker := time.NewTicker(depositProcessorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			return
		case <-t.rootCtx.Done():
			return
		case <-ticker.C:
			t.runDepositProcessorCycle()
		}
	}
}

func (t *Ton) runDepositProcessorCycle() {
	ctx := t.rootCtx

	ids, err := t.dbStorage.GetProcessingTransactionIDs(ctx)
	if err != nil {
		t.log.Errorw("deposit processor: get processing transactions", "err", err)
		return
	}
	if len(ids) == 0 {
		return
	}

	for _, txID := range ids {
		if err = t.dbStorage.ProcessDepositForTransaction(ctx, txID); err != nil {
			t.log.Errorw("deposit processor: process transaction",
				"tx_id", txID,
				"err", err,
			)
		}
	}
}
