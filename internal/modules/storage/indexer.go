package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/wickedv43/TAM-backend/internal/common/constants/transaction_status"
	"github.com/wickedv43/TAM-backend/internal/database/ent"
	"github.com/wickedv43/TAM-backend/internal/database/ent/customer"
	"github.com/wickedv43/TAM-backend/internal/database/ent/indexerstate"
	"github.com/wickedv43/TAM-backend/internal/database/ent/transaction"
)

// TransactionParams contains all parameters needed to save a transaction
type TransactionParams struct {
	Sender        string
	WalletAddress string
	AmountNano    int64   // Amount in nanoton (1 TON = 1e9 nanoton)
	AmountTON     float64 // Amount in TON (user-friendly)
	TxHash        string
	Seqno         uint64
	LT            uint64
}

// IndexerInterface defines methods for TON blockchain indexer operations.
type IndexerInterface interface {
	GetAllBounceableAddresses(ctx context.Context) ([]string, error)
	GetOrCreateIndexerState(ctx context.Context, serviceName string, initialSeqno uint64) (*ent.IndexerState, error)
	UpdateIndexerState(ctx context.Context, state *ent.IndexerState, seqno uint64) error
	SaveTransaction(ctx context.Context, params TransactionParams) error
	GetProcessingTransactionIDs(ctx context.Context) ([]uuid.UUID, error)
	ProcessDepositForTransaction(ctx context.Context, txID uuid.UUID) error
}

// GetAllBounceableAddresses returns all bounceable addresses from customers table
func (p *PostgresDB) GetAllBounceableAddresses(ctx context.Context) ([]string, error) {
	customers, err := p.db.Customer.Query().
		Select("address_bounceable").
		All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "querying customers")
	}

	addresses := make([]string, 0, len(customers))
	for _, c := range customers {
		if c.AddressBounceable != "" {
			addresses = append(addresses, c.AddressBounceable)
		}
	}

	return addresses, nil
}

// GetOrCreateIndexerState gets or creates indexer state for the given service name
func (p *PostgresDB) GetOrCreateIndexerState(ctx context.Context, serviceName string, initialSeqno uint64) (*ent.IndexerState, error) {
	state, err := p.db.IndexerState.Query().
		Where(indexerstate.ServiceName(serviceName)).
		Only(ctx)

	if ent.IsNotFound(err) {
		state, err = p.db.IndexerState.Create().
			SetServiceName(serviceName).
			SetLastSeqno(initialSeqno).
			Save(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "creating indexer state")
		}
		return state, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "querying indexer state")
	}

	return state, nil
}

// UpdateIndexerState updates the last processed seqno
func (p *PostgresDB) UpdateIndexerState(ctx context.Context, state *ent.IndexerState, seqno uint64) error {
	_, err := p.db.IndexerState.UpdateOne(state).
		SetLastSeqno(seqno).
		Save(ctx)
	return errors.Wrap(err, "updating indexer state")
}

// SaveTransaction saves a TON transaction. Skips if hash already exists (idempotency).
func (p *PostgresDB) SaveTransaction(ctx context.Context, params TransactionParams) error {
	exists, err := p.db.Transaction.Query().
		Where(transaction.TxHash(params.TxHash)).
		Exist(ctx)
	if err != nil {
		return errors.Wrap(err, "checking transaction existence")
	}
	if exists {
		return nil
	}

	_, err = p.db.Transaction.Create().
		SetSender(params.Sender).
		SetWalletAddress(params.WalletAddress).
		SetAmount(params.AmountTON).
		SetTxHash(params.TxHash).
		SetSeqno(params.Seqno).
		SetLt(params.LT).
		SetStatus(int(transaction_status.PROCESSING)).
		Save(ctx)
	if err != nil {
		return errors.Wrap(err, "saving transaction")
	}

	return nil
}

// GetProcessingTransactionIDs returns IDs of transactions with status PROCESSING.
func (p *PostgresDB) GetProcessingTransactionIDs(ctx context.Context) ([]uuid.UUID, error) {
	ids, err := p.db.Transaction.Query().
		Where(transaction.StatusEQ(int(transaction_status.PROCESSING))).
		IDs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "querying processing transactions")
	}
	return ids, nil
}

// ProcessDepositForTransaction credits customer balance and marks transaction DONE.
func (p *PostgresDB) ProcessDepositForTransaction(ctx context.Context, txID uuid.UUID) error {
	txRec, err := p.db.Transaction.Get(ctx, txID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "getting transaction")
	}
	if txRec.Status != int(transaction_status.PROCESSING) {
		return nil // already processed or not processing
	}

	cust, err := p.db.Customer.Query().
		Where(
			customer.Or(
				customer.AddressNonbounceableEQ(txRec.WalletAddress),
				customer.AddressBounceableEQ(txRec.WalletAddress),
			),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "finding customer by wallet address")
	}

	if txRec.Amount <= 0 {
		_, err = p.db.Transaction.UpdateOneID(txID).
			SetStatus(int(transaction_status.DONE)).
			Save(ctx)
		return errors.Wrap(err, "marking transaction DONE")
	}

	dbTx, err := p.db.Tx(ctx)
	if err != nil {
		return errors.Wrap(err, "starting db transaction")
	}
	defer func() {
		if v := recover(); v != nil {
			_ = dbTx.Rollback()
		}
	}()

	_, err = dbTx.Customer.UpdateOneID(cust.ID).
		AddTonBalance(txRec.Amount).
		Save(ctx)
	if err != nil {
		_ = dbTx.Rollback()
		return errors.Wrap(err, "adding ton_balance")
	}

	_, err = dbTx.Transaction.UpdateOneID(txID).
		SetStatus(int(transaction_status.DONE)).
		Save(ctx)
	if err != nil {
		_ = dbTx.Rollback()
		return errors.Wrap(err, "setting transaction DONE")
	}

	return dbTx.Commit()
}
