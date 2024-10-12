package duties

import (
	"context"

	"github.com/ssvlabs/ssv/networkconfig"
	"github.com/ssvlabs/ssv/operator/slotoracle"
	"go.uber.org/zap"
)

//go:generate mockgen -package=duties -destination=./base_handler_mock.go -source=./base_handler.go

type dutyHandler interface {
	Setup(
		name string,
		logger *zap.Logger,
		beaconNode BeaconNode,
		executionClient ExecutionClient,
		network networkconfig.NetworkConfig,
		validatorProvider ValidatorProvider,
		validatorController ValidatorController,
		dutiesExecutor DutiesExecutor,
		slotOracleProvider slotoracle.Provider,
		reorgEvents chan ReorgEvent,
		indicesChange chan struct{},
	)
	HandleDuties(context.Context)
	HandleInitialDuties(context.Context)
	Name() string
}

type baseHandler struct {
	logger              *zap.Logger
	beaconNode          BeaconNode
	executionClient     ExecutionClient
	network             networkconfig.NetworkConfig
	validatorProvider   ValidatorProvider
	validatorController ValidatorController
	dutiesExecutor      DutiesExecutor
	ticker              slotoracle.SlotOracle

	reorg         chan ReorgEvent
	indicesChange chan struct{}

	indicesChanged bool
}

func (h *baseHandler) Setup(
	name string,
	logger *zap.Logger,
	beaconNode BeaconNode,
	executionClient ExecutionClient,
	network networkconfig.NetworkConfig,
	validatorProvider ValidatorProvider,
	validatorController ValidatorController,
	dutiesExecutor DutiesExecutor,
	slotOracleProvider slotoracle.Provider,
	reorgEvents chan ReorgEvent,
	indicesChange chan struct{},
) {
	h.logger = logger.With(zap.String("handler", name))
	h.beaconNode = beaconNode
	h.executionClient = executionClient
	h.network = network
	h.validatorProvider = validatorProvider
	h.validatorController = validatorController
	h.dutiesExecutor = dutiesExecutor
	h.ticker = slotOracleProvider()
	h.reorg = reorgEvents
	h.indicesChange = indicesChange
}

func (h *baseHandler) warnMisalignedSlotAndDuty(dutyType string) {
	h.logger.Debug("current slot and duty slot are not aligned, "+
		"assuming diff caused by a time drift - ignoring and executing duty", zap.String("type", dutyType))
}

func (h *baseHandler) HandleInitialDuties(context.Context) {
	// Do nothing
}
