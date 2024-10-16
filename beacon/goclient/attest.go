package goclient

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/api"
	eth2apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// AttesterDuties returns attester duties for a given epoch.
func (gc *GoClient) AttesterDuties(ctx context.Context, epoch phase0.Epoch, validatorIndices []phase0.ValidatorIndex) ([]*eth2apiv1.AttesterDuty, error) {
	resp, err := gc.client.AttesterDuties(ctx, &api.AttesterDutiesOpts{
		Epoch:   epoch,
		Indices: validatorIndices,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to obtain attester duties: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("attester duties response is nil")
	}

	return resp.Data, nil
}

// GetAttestationData returns attestation data for a given slot (which is same for all 64 committeeIndex
// values that identify Ethereum committees chosen to attest on this slot).
// Note, committeeIndex is an optional parameter that will be used to set AttestationData.Index
// in the resulting data returned from this function.
// Note, result returned is meant to be read-only, it's not safe to modify it (because it will be
// accessed by multiple concurrent readers).
func (gc *GoClient) GetAttestationData(slot phase0.Slot, committeeIndex phase0.CommitteeIndex) (
	result *phase0.AttestationData,
	version spec.DataVersion,
	err error,
) {
	// Final processing for result returned.
	defer func() {
		if err != nil {
			// Nothing to process, just propagate error.
			return
		}

		// Assign committeeIndex passed to GetAttestationData call, the rest of attestation data stays
		// unchanged.
		// Note, we cannot return result object directly here modifying its Index value because it
		// would be unsynchronised concurrent write (since it's cached for other concurrent readers
		// to access). Hence, we return shallow copy here. We don't need to return deep copy because
		// the callers of GetAttestationData will only use this data to read it (they won't update it).
		result = &phase0.AttestationData{
			Slot:            result.Slot,
			Index:           committeeIndex,
			BeaconBlockRoot: result.BeaconBlockRoot,
			Source:          result.Source,
			Target:          result.Target,
		}
	}()

	// Check cache.
	cachedResult, ok := gc.attestationDataCache.Get(slot)
	if ok {
		return cachedResult, spec.DataVersionPhase0, nil
	}

	// Have to make beacon node request and cache the result.
	attDataReqStart := time.Now()
	result, err = func() (*phase0.AttestationData, error) {
		// Requests with the same slot number must lock the same mutex to avoid sending duplicate requests.
		reqMu := &gc.attestationReqMuPool[int64(slot)%int64(len(gc.attestationReqMuPool))]
		reqMu.Lock()
		defer reqMu.Unlock()

		// Prevent making more than 1 beacon node requests in case somebody has already made this
		// request concurrently and succeeded while we were waiting.
		cachedResult, ok := gc.attestationDataCache.Get(slot)
		if ok {
			return cachedResult, nil
		}

		resp, err := gc.client.AttestationData(gc.ctx, &api.AttestationDataOpts{
			Slot: slot,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get attestation data: %w", err)
		}
		if resp == nil {
			return nil, fmt.Errorf("attestation data response is nil")
		}

		gc.attestationDataCache.Set(slot, resp.Data)
		// gc.recentAttestationSlot doesn't need to be the latest slot we've processed, it just needs
		// to gradually go up over time (so we know what attestation data in our cache is no longer relevant).
		gc.recentAttestationSlot.Store(uint64(slot))

		return resp.Data, nil
	}()
	metricsAttesterDataRequest.Observe(time.Since(attDataReqStart).Seconds())
	if err != nil {
		return nil, DataVersionNil, err
	}

	return result, spec.DataVersionPhase0, nil
}

// pruneStaleAttestationDataRunner will periodically prune attestationDataCache to keep it from growing
// perpetually.
func (gc *GoClient) pruneStaleAttestationDataRunner() {
	pruneStaleAttestationData := func() {
		// slotRetainCnt defines how many recent slots we want to preserve in attestation data cache.
		slotRetainCnt := 5 * gc.network.SlotsPerEpoch()
		gc.attestationDataCache.Range(func(slot phase0.Slot, data *phase0.AttestationData) bool {
			if uint64(slot) < (gc.recentAttestationSlot.Load() - slotRetainCnt) {
				gc.attestationDataCache.Delete(slot)
			}
			return true
		})
	}

	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-gc.ctx.Done():
			return
		case <-ticker.C:
			pruneStaleAttestationData()
		}
	}
}

// SubmitAttestations implements Beacon interface
func (gc *GoClient) SubmitAttestations(attestations []*phase0.Attestation) error {
	return gc.client.SubmitAttestations(gc.ctx, attestations)
}
