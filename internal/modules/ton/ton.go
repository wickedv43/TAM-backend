package ton

import (
	"context"
	"sync"

	"github.com/wickedv43/TAM-backend/internal/config"
	"github.com/wickedv43/TAM-backend/internal/logger"
	"github.com/wickedv43/TAM-backend/internal/modules/storage"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"

	"github.com/samber/do/v2"
)

type Ton struct {
	api ton.APIClientWrapped
	cfg *config.Config
	log *zap.SugaredLogger

	dbStorage storage.IndexerInterface

	rootCtx context.Context

	addressBook *SafeAddressBook
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

func NewTon(i do.Injector) (*Ton, error) {
	cfg := do.MustInvoke[*config.Config](i)
	log := do.MustInvoke[*logger.Logger](i).Named("ton")
	dbStorage := do.MustInvoke[*storage.PostgresDB](i)

	client := liteclient.NewConnectionPool()

	tonCfg, err := liteclient.GetConfigFromUrl(context.Background(), cfg.Ton.MainNet)
	if err != nil {
		return nil, errors.Wrap(err, "getting config from url")
	}

	// Connect to mainnet lite servers
	if err = client.AddConnectionsFromConfig(context.Background(), tonCfg); err != nil {
		return nil, errors.Wrap(err, "adding connections from config")
	}

	api := ton.NewAPIClient(client, ton.ProofCheckPolicyFast).WithRetry()
	api.SetTrustedBlockFromConfig(tonCfg)

	rootCtx, err := do.InvokeNamed[context.Context](i, "root.context")
	if err != nil {
		return nil, errors.Wrap(err, "root.context")
	}

	return &Ton{
		api:       api,
		cfg:       cfg,
		log:       log,
		dbStorage: dbStorage,
		rootCtx:   client.StickyContext(rootCtx),
		stopCh:    make(chan struct{}),
	}, nil
}

// Start launches the background indexing process
func (t *Ton) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		t.log.Info("ton indexer service stopped")
		close(t.stopCh)
	}()

	if !t.cfg.Indexer {
		t.log.Warn("ton indexer service is OFF")
		return nil
	}

	t.log.Info("starting ton indexer service")
	t.wg.Add(3)

	go t.indexerLoop()
	go t.addressBookRefresher()
	go t.depositProcessorLoop()

	return nil
}
