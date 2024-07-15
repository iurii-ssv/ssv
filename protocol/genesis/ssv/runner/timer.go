package runner

import (
	specqbft "github.com/ssvlabs/ssv-spec-pre-cc/qbft"
	spectypes "github.com/ssvlabs/ssv-spec-pre-cc/types"
	"go.uber.org/zap"

	"github.com/ssvlabs/ssv/protocol/genesis/qbft/instance"
	"github.com/ssvlabs/ssv/protocol/genesis/qbft/roundtimer"
)

type TimeoutF func(logger *zap.Logger, identifier spectypes.MessageID, height specqbft.Height) roundtimer.OnRoundTimeoutF

func (b *BaseRunner) registerTimeoutHandler(logger *zap.Logger, instance *instance.Instance, height specqbft.Height) {
	identifier := spectypes.MessageIDFromBytes(instance.State.ID)
	timer, ok := instance.GetConfig().GetTimer().(*roundtimer.RoundTimer)
	if ok {
		timer.OnTimeout(b.TimeoutF(logger, identifier, height))
	}
}