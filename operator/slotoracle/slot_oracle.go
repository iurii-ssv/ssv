package slotoracle

import (
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ssvlabs/ssv/utils/casts"
	"go.uber.org/zap"
)

//go:generate mockgen -package=mocks -destination=./mocks/slot_oracle.go -source=./slot_oracle.go

type Provider func() SlotOracle

// SlotOracle provides a way to keep track of Ethereum slots as they change over time.
type SlotOracle interface {
	// Next advances SlotOracle slot number (which keeps track of the "freshest" slot value
	// from Ethereum perspective) potentially jumping several slots ahead. It returns a channel
	// that will relay 1 tick signalling that "freshest" slot has started.
	// Note: The caller is RESPONSIBLE for calling Next method periodically in order for
	// SlotOracle to advance forward (to keep ticking) to newer slots.
	Next() <-chan time.Time
	// Slot returns the next slot number SlotOracle will tick on.
	// Note: The caller is RESPONSIBLE for calling Next method periodically in order for
	// SlotOracle to advance forward (to keep ticking).
	Slot() phase0.Slot
}

type Config struct {
	SlotDuration time.Duration
	GenesisTime  time.Time
}

// slotOracle implements SlotOracle.
// Note: this implementation is NOT THREAD-SAFE, hence it all its methods should be called
// in a serialized fashion (concurrent calls can result in unexpected behavior).
type slotOracle struct {
	logger       *zap.Logger
	timer        Timer
	slotDuration time.Duration
	genesisTime  time.Time
	slot         phase0.Slot
}

// New returns a goroutine-free SlotOracle implementation which is not thread-safe.
func New(logger *zap.Logger, cfg Config) *slotOracle {
	return newWithCustomTimer(logger, cfg, NewTimer)
}

func newWithCustomTimer(logger *zap.Logger, cfg Config, timerProvider TimerProvider) *slotOracle {
	timeSinceGenesis := time.Since(cfg.GenesisTime)

	var (
		initialDelay time.Duration
		initialSlot  phase0.Slot
	)
	if timeSinceGenesis < 0 {
		// Genesis time is in the future
		initialDelay = -timeSinceGenesis // Wait until the genesis time
		initialSlot = phase0.Slot(0)     // Start at slot 0
	} else {
		slotsSinceGenesis := timeSinceGenesis / cfg.SlotDuration
		nextSlotStartTime := cfg.GenesisTime.Add((slotsSinceGenesis + 1) * cfg.SlotDuration)
		initialDelay = time.Until(nextSlotStartTime)
		initialSlot = phase0.Slot(slotsSinceGenesis)
	}

	return &slotOracle{
		logger:       logger,
		timer:        timerProvider(initialDelay),
		slotDuration: cfg.SlotDuration,
		genesisTime:  cfg.GenesisTime,
		slot:         initialSlot,
	}
}

// Next implements SlotOracle.Next.
// Note: this method is not thread-safe.
func (s *slotOracle) Next() <-chan time.Time {
	timeSinceGenesis := time.Since(s.genesisTime)
	if timeSinceGenesis < 0 {
		// We are waiting for slotOracle to tick at s.genesisTime (signalling 0th slot start).
		return s.timer.C()
	}
	if !s.timer.Stop() {
		// try to drain the channel, but don't block if there's no value
		select {
		case <-s.timer.C():
		default:
		}
	}
	nextSlot := phase0.Slot(timeSinceGenesis/s.slotDuration) + 1 // #nosec G115
	if nextSlot <= s.slot {
		// We've already ticked for this slot, so we need to wait for the next one.
		nextSlot = s.slot + 1
		s.logger.Debug("slotOracle: double tick", zap.Uint64("slot", uint64(s.slot)))
	}
	nextSlotStartTime := s.genesisTime.Add(casts.DurationFromUint64(uint64(nextSlot)) * s.slotDuration)
	s.timer.Reset(time.Until(nextSlotStartTime))
	s.slot = nextSlot
	return s.timer.C()
}

// Slot implements SlotOracle.Slot.
// Note: this method is not thread-safe.
func (s *slotOracle) Slot() phase0.Slot {
	return s.slot
}
