package ton

import (
	"context"
	"encoding/base64"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/modules/storage"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
)

const (
	indexerServiceName = "main_indexer"
	blockCheckInterval = 5 * time.Second
	// @TODO: add address immediately after customer creation (not list refresh)
	addressRefreshInterval = 5 * time.Second
	txBatchSize            = 100
)

// SafeAddressBook caches customer wallets in memory to avoid DB hits on every tx check
type SafeAddressBook struct {
	mu    sync.RWMutex
	addrs map[string]struct{}
}

func (s *SafeAddressBook) Has(addr string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.addrs[addr]
	return ok
}

func (s *SafeAddressBook) LoadFromDB(ctx context.Context, storage storage.IndexerInterface) error {
	addresses, err := storage.GetAllBounceableAddresses(ctx)
	if err != nil {
		return err
	}

	newMap := make(map[string]struct{}, len(addresses))
	for _, addr := range addresses {
		newMap[addr] = struct{}{}
	}

	s.mu.Lock()
	s.addrs = newMap
	s.mu.Unlock()
	return nil
}

func (s *SafeAddressBook) GetAddresses() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	addrs := make([]string, 0, len(s.addrs))
	for addr := range s.addrs {
		addrs = append(addrs, addr)
	}
	return addrs
}

func (t *Ton) indexerLoop() {
	defer t.wg.Done()

	// Initialize address book
	t.addressBook = &SafeAddressBook{addrs: make(map[string]struct{})}
	if err := t.addressBook.LoadFromDB(t.rootCtx, t.dbStorage); err != nil {
		t.log.Errorw("failed to load initial address book", "err", err)
	} else {
		t.log.Infow("address book loaded",
			"addresses_count", len(t.addressBook.addrs),
		)
	}

	// log initial state
	state, err := t.getOrCreateState(t.rootCtx)
	if err != nil {
		t.log.Errorw("failed to get initial state", "err", err)
	} else {
		t.log.Infow("indexer started", "last_seqno", state.LastSeqno)
	}

	ticker := time.NewTicker(blockCheckInterval)
	defer ticker.Stop()

	totalBlocksProcessed := 0
	startTime := time.Now()

	t.processBlocksCycle(&totalBlocksProcessed, startTime)

	for {
		select {
		case <-t.stopCh:
			return
		case <-t.rootCtx.Done():
			return
		case <-ticker.C:
			t.processBlocksCycle(&totalBlocksProcessed, startTime)
		}
	}
}

func (t *Ton) processBlocksCycle(totalBlocksProcessed *int, startTime time.Time) {
	blocksProcessed, txCount, err := t.processAllPendingBlocks(t.rootCtx)
	if err != nil {
		t.log.Errorw("failed to process blocks", "err", err)
		return
	}

	*totalBlocksProcessed += blocksProcessed

	if blocksProcessed > 0 {
		elapsed := time.Since(startTime)
		blocksPerSec := float64(*totalBlocksProcessed) / elapsed.Seconds()
		t.log.Infow("cycle completed",
			"blocks_in_cycle", blocksProcessed,
			"tx_in_cycle", txCount,
			"total_blocks_processed", *totalBlocksProcessed,
			"elapsed", elapsed.Round(time.Second).String(),
			"blocks_per_sec", blocksPerSec,
		)
	}
}

func (t *Ton) addressBookRefresher() {
	ticker := time.NewTicker(addressRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			if err := t.addressBook.LoadFromDB(t.rootCtx, t.dbStorage); err != nil {
				t.log.Errorw("failed to refresh address book", "err", err)
			} else {
				t.log.Infow("address book refreshed", "addresses_count", len(t.addressBook.addrs))
			}
		}
	}
}

// processAllPendingBlocks processes all blocks from lastSeqno to current
// and saves the block number to DB only at the end
func (t *Ton) processAllPendingBlocks(ctx context.Context) (blocksProcessed int, totalTxCount int, err error) {
	master, err := t.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return 0, 0, errors.Wrap(err, "getting masterchain info")
	}

	state, err := t.getOrCreateState(ctx)
	if err != nil {
		return 0, 0, errors.Wrap(err, "getting indexer state")
	}
	lastSeqno := state.LastSeqno
	currentSeqno := uint64(master.SeqNo)

	if lastSeqno >= currentSeqno {
		return 0, 0, nil
	}

	blocksToProcess := currentSeqno - lastSeqno
	t.log.Infow("starting block processing cycle",
		"from_seqno", lastSeqno+1,
		"to_seqno", currentSeqno,
		"blocks_to_process", blocksToProcess,
	)

	for seqno := lastSeqno + 1; seqno <= currentSeqno; seqno++ {
		select {
		case <-ctx.Done():
			if blocksProcessed > 0 {
				_ = t.dbStorage.UpdateIndexerState(ctx, state, seqno-1)
			}
			return blocksProcessed, totalTxCount, ctx.Err()
		case <-t.stopCh:
			if blocksProcessed > 0 {
				_ = t.dbStorage.UpdateIndexerState(ctx, state, seqno-1)
			}
			return blocksProcessed, totalTxCount, nil
		default:
		}

		txCount, err := t.processBlock(ctx, master, seqno)
		if err != nil {
			if blocksProcessed > 0 {
				_ = t.dbStorage.UpdateIndexerState(ctx, state, seqno-1)
			}
			return blocksProcessed, totalTxCount, errors.Wrapf(err, "processing block %d", seqno)
		}

		blocksProcessed++
		totalTxCount += txCount
	}

	if blocksProcessed > 0 {
		if err := t.dbStorage.UpdateIndexerState(ctx, state, currentSeqno); err != nil {
			return blocksProcessed, totalTxCount, errors.Wrap(err, "saving indexer state")
		}
	}

	return blocksProcessed, totalTxCount, nil
}

func (t *Ton) processBlock(ctx context.Context, master *ton.BlockIDExt, seqno uint64) (txCount int, err error) {
	fullBlockID, err := t.api.LookupBlock(ctx, master.Workchain, master.Shard, uint32(seqno))
	if err != nil {
		return 0, errors.Wrap(err, "lookup block")
	}

	txCount, err = t.processMasterBlock(ctx, fullBlockID)
	if err != nil {
		return 0, errors.Wrap(err, "processing master block")
	}

	return txCount, nil
}

func (t *Ton) processMasterBlock(ctx context.Context, master *ton.BlockIDExt) (totalTxCount int, err error) {
	shards, err := t.api.GetBlockShardsInfo(ctx, master)
	if err != nil {
		return 0, errors.Wrap(err, "getting shards info")
	}

	t.log.Debugw("processing master block", "shards_count", len(shards))

	for i, shard := range shards {
		shardStartTime := time.Now()
		txCount, err := t.processShardBlock(ctx, shard)
		if err != nil {
			return totalTxCount, errors.Wrapf(err, "processing shard block %d/%d", i+1, len(shards))
		}
		totalTxCount += txCount
		shardDuration := time.Since(shardStartTime)

		if shardDuration > time.Second {
			t.log.Warnw("slow shard processing",
				"shard_index", i+1,
				"shards_total", len(shards),
				"tx_count", txCount,
				"duration_ms", shardDuration.Milliseconds(),
			)
		}
	}
	return totalTxCount, nil
}

func (t *Ton) processShardBlock(ctx context.Context, shard *ton.BlockIDExt) (txCount int, err error) {
	var afterTx *ton.TransactionID3
	batchCount := 0
	totalRequestTime := time.Duration(0)
	totalProcessTime := time.Duration(0)
	relevantTxCount := 0

	maxRequestDuration := 2 * time.Minute

	for {
		batchStartTime := time.Now()
		batchCount++

		reqCtx, cancel := context.WithTimeout(ctx, maxRequestDuration)
		txs, more, err := t.api.GetBlockTransactionsV2(reqCtx, shard, txBatchSize, afterTx)
		cancel()

		requestDuration := time.Since(batchStartTime)
		totalRequestTime += requestDuration

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				t.log.Errorw("transaction request timeout",
					"shard_workchain", shard.Workchain,
					"txs_processed_so_far", txCount,
					"batch_count", batchCount,
					"request_duration_ms", requestDuration.Milliseconds(),
					"total_request_time_ms", totalRequestTime.Milliseconds(),
				)
				return txCount, errors.Wrap(err, "getting block transactions (timeout)")
			}
			return txCount, errors.Wrap(err, "getting block transactions")
		}

		if len(txs) == 0 {
			break
		}

		txCount += len(txs)

		processStartTime := time.Now()
		batchRelevantCount := 0

		for _, shortTx := range txs {
			addr := address.NewAddress(0, byte(shard.Workchain), shortTx.Account)
			addrStr := addr.Bounce(true).String()

			if !t.addressBook.Has(addrStr) {
				continue
			}

			t.log.Infow("found matching address",
				"addr", addrStr,
				"lt", shortTx.LT,
			)

			batchRelevantCount++
			relevantTxCount++

			if err := t.checkTransaction(ctx, shard, addr, shortTx.LT, uint64(shard.SeqNo)); err != nil {
				t.log.Errorw("failed to check transaction",
					"addr", addrStr,
					"err", err,
				)
			}
		}

		processDuration := time.Since(processStartTime)
		totalProcessTime += processDuration

		if len(txs) > 0 {
			lastTx := txs[len(txs)-1]
			afterTx = &ton.TransactionID3{
				Account: lastTx.Account,
				LT:      lastTx.LT,
			}
		}

		if !more {
			break
		}

		if ctx.Err() != nil {
			return txCount, errors.Wrap(ctx.Err(), "block processing timeout exceeded")
		}
	}

	return txCount, nil
}

func (t *Ton) checkTransaction(ctx context.Context, block *ton.BlockIDExt, addr *address.Address, lt uint64, seqno uint64) error {
	txStartTime := time.Now()
	tx, err := t.api.GetTransaction(ctx, block, addr, lt)
	if err != nil {
		return errors.Wrap(err, "getting transaction details")
	}
	txFetchDuration := time.Since(txStartTime)

	if tx.IO.In == nil || tx.IO.In.MsgType != tlb.MsgTypeInternal {
		return nil
	}

	msg := tx.IO.In.AsInternal()
	if msg.Amount.Nano().Sign() <= 0 {
		return nil
	}

	if msg.Bounce && msg.Bounced {
		return nil
	}

	senderAddr := msg.SrcAddr.String()
	destAddr := msg.DstAddr.String()
	txHash := base64.StdEncoding.EncodeToString(tx.Hash)
	amountNano := msg.Amount.Nano().Int64()
	amountTON := float64(amountNano) / 1e9

	if txFetchDuration > 1*time.Second {
		t.log.Warnw("slow transaction fetch",
			"addr", destAddr,
			"duration_ms", txFetchDuration.Milliseconds(),
		)
	}

	t.log.Infow("TON DEPOSIT DETECTED",
		"sender", senderAddr,
		"address", destAddr,
		"amount_ton", amountTON,
		"tx_hash", txHash,
		"seqno", seqno,
		"lt", tx.LT,
	)

	depositStartTime := time.Now()
	params := storage.TransactionParams{
		Sender:        senderAddr,
		WalletAddress: destAddr,
		AmountNano:    amountNano,
		AmountTON:     amountTON,
		TxHash:        txHash,
		Seqno:         seqno,
		LT:            tx.LT,
	}
	if err = t.dbStorage.SaveTransaction(ctx, params); err != nil {
		saveDuration := time.Since(depositStartTime)
		t.log.Errorw("failed to save transaction",
			"addr", destAddr,
			"tx_hash", txHash,
			"duration_ms", saveDuration.Milliseconds(),
			"err", err,
		)
		return errors.Wrap(err, "saving transaction")
	}
	saveDuration := time.Since(depositStartTime)

	if saveDuration > 500*time.Millisecond {
		t.log.Warnw("slow transaction save",
			"addr", destAddr,
			"duration_ms", saveDuration.Milliseconds(),
		)
	}

	return nil
}

func (t *Ton) getOrCreateState(ctx context.Context) (*ent.IndexerState, error) {
	master, err := t.api.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting initial masterchain info")
	}

	state, err := t.dbStorage.GetOrCreateIndexerState(ctx, indexerServiceName, uint64(master.SeqNo))
	if err != nil {
		return nil, errors.Wrap(err, "getting indexer state")
	}

	return state, nil
}
