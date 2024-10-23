package operator

import (
	"context"
	"fmt"
	"go.uber.org/zap"

	"github.com/ssvlabs/ssv/eth/executionclient"
	"github.com/ssvlabs/ssv/exporter/api"
	qbftstorage "github.com/ssvlabs/ssv/ibft/storage"
	"github.com/ssvlabs/ssv/logging"
	"github.com/ssvlabs/ssv/logging/fields"
	"github.com/ssvlabs/ssv/network"
	"github.com/ssvlabs/ssv/networkconfig"
	"github.com/ssvlabs/ssv/operator/duties"
	"github.com/ssvlabs/ssv/operator/duties/dutystore"
	"github.com/ssvlabs/ssv/operator/fee_recipient"
	"github.com/ssvlabs/ssv/operator/slotticker"
	"github.com/ssvlabs/ssv/operator/storage"
	"github.com/ssvlabs/ssv/operator/validator"
	beaconprotocol "github.com/ssvlabs/ssv/protocol/v2/blockchain/beacon"
	storage2 "github.com/ssvlabs/ssv/registry/storage"
	"github.com/ssvlabs/ssv/storage/basedb"
)

// Node represents the behavior of SSV node
type Node interface {
	Start(logger *zap.Logger) error
}

// Options contains options to create the node
type Options struct {
	// NetworkName is the network name of this node
	NetworkName         string `yaml:"Network" env:"NETWORK" env-default:"mainnet" env-description:"Network is the network of this node"`
	CustomDomainType    string `yaml:"CustomDomainType" env:"CUSTOM_DOMAIN_TYPE" env-default:"" env-description:"Override the SSV domain type. This is used to isolate the node from the rest of the network. Do not set unless you know what you are doing. Example: 0x01020304"`
	Network             networkconfig.NetworkConfig
	BeaconNode          beaconprotocol.BeaconNode // TODO: consider renaming to ConsensusClient
	ExecutionClient     *executionclient.ExecutionClient
	P2PNetwork          network.P2PNetwork
	Context             context.Context
	DB                  basedb.Database
	ValidatorController validator.Controller
	ValidatorStore      storage2.ValidatorStore
	ValidatorOptions    validator.ControllerOptions `yaml:"ValidatorOptions"`
	DutyStore           *dutystore.Store
	WS                  api.WebSocketServer
	WsAPIPort           int
	Metrics             nodeMetrics
}

// operatorNode implements Node interface
type operatorNode struct {
	network          networkconfig.NetworkConfig
	context          context.Context
	validatorsCtrl   validator.Controller
	validatorOptions validator.ControllerOptions
	consensusClient  beaconprotocol.BeaconNode
	executionClient  *executionclient.ExecutionClient
	net              network.P2PNetwork
	storage          storage.Storage
	qbftStorage      *qbftstorage.QBFTStores
	dutyScheduler    *duties.Scheduler
	feeRecipientCtrl fee_recipient.RecipientController

	ws        api.WebSocketServer
	wsAPIPort int

	metrics nodeMetrics
}

// New is the constructor of operatorNode
func New(logger *zap.Logger, opts Options, slotTickerProvider slotticker.Provider, qbftStorage *qbftstorage.QBFTStores) Node {
	node := &operatorNode{
		context:          opts.Context,
		validatorsCtrl:   opts.ValidatorController,
		validatorOptions: opts.ValidatorOptions,
		network:          opts.Network,
		consensusClient:  opts.BeaconNode,
		executionClient:  opts.ExecutionClient,
		net:              opts.P2PNetwork,
		storage:          opts.ValidatorOptions.RegistryStorage,
		qbftStorage:      qbftStorage,
		dutyScheduler: duties.NewScheduler(&duties.SchedulerOptions{
			Ctx:                 opts.Context,
			BeaconNode:          opts.BeaconNode,
			ExecutionClient:     opts.ExecutionClient,
			Network:             opts.Network,
			ValidatorProvider:   opts.ValidatorStore.WithOperatorID(opts.ValidatorOptions.OperatorDataStore.GetOperatorID),
			ValidatorController: opts.ValidatorController,
			DutyExecutor:        opts.ValidatorController,
			IndicesChg:          opts.ValidatorController.IndicesChangeChan(),
			ValidatorExitCh:     opts.ValidatorController.ValidatorExitChan(),
			DutyStore:           opts.DutyStore,
			SlotTickerProvider:  slotTickerProvider,
			P2PNetwork:          opts.P2PNetwork,
		}),
		feeRecipientCtrl: fee_recipient.NewController(&fee_recipient.ControllerOptions{
			Ctx:                opts.Context,
			BeaconClient:       opts.BeaconNode,
			Network:            opts.Network,
			ShareStorage:       opts.ValidatorOptions.RegistryStorage.Shares(),
			RecipientStorage:   opts.ValidatorOptions.RegistryStorage,
			OperatorDataStore:  opts.ValidatorOptions.OperatorDataStore,
			SlotTickerProvider: slotTickerProvider,
		}),

		ws:        opts.WS,
		wsAPIPort: opts.WsAPIPort,

		metrics: opts.Metrics,
	}

	if node.metrics == nil {
		node.metrics = nopMetrics{}
	}

	return node
}

// Start starts to stream duties and run IBFT instances
func (n *operatorNode) Start(logger *zap.Logger) error {
	logger.Named(logging.NameOperator)

	logger.Info("All required services are ready. OPERATOR SUCCESSFULLY CONFIGURED AND NOW RUNNING!")

	go func() {
		err := n.startWSServer(logger)
		if err != nil {
			// TODO: think if we need to panic
			return
		}
	}()

	// Start the duty scheduler, and a background goroutine to crash the node
	// in case there were any errors.
	if err := n.dutyScheduler.Start(n.context, logger); err != nil {
		return fmt.Errorf("failed to run duty scheduler: %w", err)
	}

	n.validatorsCtrl.StartNetworkHandlers()

	if n.validatorOptions.Exporter {
		// Subscribe to all subnets.
		err := n.net.SubscribeAll(logger)
		if err != nil {
			logger.Error("failed to subscribe to all subnets", zap.Error(err))
		}
	}
	go n.net.UpdateSubnets(logger)
	go n.net.UpdateScoreParams(logger)
	n.validatorsCtrl.ForkListener(logger)
	n.validatorsCtrl.StartValidators()
	go n.reportOperators(logger)

	go n.feeRecipientCtrl.Start(logger)
	go n.validatorsCtrl.UpdateValidatorMetaDataLoop()

	//// TODO
	//fmt.Println(fmt.Sprintf("TODO: GENERATING PRIV key"))
	//privKey, err := keys.GeneratePrivateKey()
	//if err != nil {
	//	logger.Fatal("failed to generate private key", zap.Error(err))
	//}
	//// msg we'll use for benchmarking.
	//msg := []byte("Some message example to be hashed for benchmarking, let's make it at " +
	//	"least 100 bytes long so we can benchmark against somewhat real-world message size. " +
	//	"Although it still might be way too small. Lajfklflfaslfjsalfalsfjsla fjlajlfaslkfjaslkf" +
	//	"lasjflkasljfLFSJLfjsalfjaslfLKFsalfjalsfjalsfjaslfjaslfjlasflfslafasklfjsalfj;eqwfgh442")
	//// msgHash is a typical sha256 hash, we use it for benchmarking because it's the most
	//// common type of data we work with.
	//msgHash := func() []byte {
	//	hash := sha256.Sum256(msg)
	//	return hash[:]
	//}()
	//result, err := privKey.Sign(msgHash)
	//if err != nil {
	//	logger.Fatal("failed to Sign message", zap.Error(err))
	//}
	//fmt.Println(fmt.Sprintf("Signed message: %s", result))
	//// TODO
	//storedConfig, foundConfig, err := n.storage.GetConfig(nil)
	//if err != nil {
	//	return fmt.Errorf("failed to get stored config: %w", err)
	//}
	//if !foundConfig {
	//	return fmt.Errorf("failed to get stored config: not found")
	//}
	//fmt.Println(fmt.Sprintf("TODO: %s", spew.Sdump(storedConfig)))
	//// TODO
	//result, err = privKey.Public().Encrypt(msgHash)
	//if err != nil {
	//	logger.Fatal("failed to Encrypt message", zap.Error(err))
	//}
	//fmt.Println(fmt.Sprintf("Encrypted message: %s", result))
	//// TODO

	if err := n.dutyScheduler.Wait(); err != nil {
		logger.Fatal("duty scheduler exited with error", zap.Error(err))
	}

	return nil
}

// HealthCheck returns a list of issues regards the state of the operator node
func (n *operatorNode) HealthCheck() error {
	// TODO: previously this checked availability of consensus & execution clients.
	// However, currently the node crashes when those clients are down,
	// so this health check is currently a positive no-op.
	return nil
}

// handleQueryRequests waits for incoming messages and
func (n *operatorNode) handleQueryRequests(logger *zap.Logger, nm *api.NetworkMessage) {
	if nm.Err != nil {
		nm.Msg = api.Message{Type: api.TypeError, Data: []string{"could not parse network message"}}
	}
	logger.Debug("got incoming export request",
		zap.String("type", string(nm.Msg.Type)))
	switch nm.Msg.Type {
	case api.TypeDecided:
		api.HandleParticipantsQuery(logger, n.qbftStorage, nm, n.network.DomainType())
	case api.TypeError:
		api.HandleErrorQuery(logger, nm)
	default:
		api.HandleUnknownQuery(logger, nm)
	}
}

func (n *operatorNode) startWSServer(logger *zap.Logger) error {
	if n.ws != nil {
		logger.Info("starting WS server")

		n.ws.UseQueryHandler(n.handleQueryRequests)

		if err := n.ws.Start(logger, fmt.Sprintf(":%d", n.wsAPIPort)); err != nil {
			return err
		}
	}

	return nil
}

func (n *operatorNode) reportOperators(logger *zap.Logger) {
	operators, err := n.storage.ListOperators(nil, 0, 1000) // TODO more than 1000?
	if err != nil {
		logger.Warn("failed to get all operators for reporting", zap.Error(err))
		return
	}
	logger.Debug("reporting operators", zap.Int("count", len(operators)))
	for i := range operators {
		n.metrics.OperatorPublicKey(operators[i].ID, operators[i].PublicKey)
		logger.Debug("report operator public key",
			fields.OperatorID(operators[i].ID),
			fields.PubKey(operators[i].PublicKey))
	}
}
