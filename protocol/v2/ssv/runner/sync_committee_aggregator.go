package runner

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	genesisspectypes "github.com/bloxapp/ssv-spec-genesis/types"
	specqbft "github.com/bloxapp/ssv-spec/qbft"
	specssv "github.com/bloxapp/ssv-spec/ssv"
	spectypes "github.com/bloxapp/ssv-spec/types"
	ssz "github.com/ferranbt/fastssz"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bloxapp/ssv/protocol/v2/qbft/controller"
	"github.com/bloxapp/ssv/protocol/v2/ssv/runner/metrics"
	ssvtypes "github.com/bloxapp/ssv/protocol/v2/types"
)

type SyncCommitteeAggregatorRunner struct {
	BaseRunner *BaseRunner

	beacon         specssv.BeaconNode
	network        specssv.Network
	signer         genesisspectypes.KeyManager
	beaconSigner   spectypes.BeaconSigner
	operatorSigner spectypes.OperatorSigner
	valCheck       specqbft.ProposedValueCheckF

	metrics metrics.ConsensusMetrics
}

func NewSyncCommitteeAggregatorRunner(
	beaconNetwork spectypes.BeaconNetwork,
	shares *map[phase0.ValidatorIndex]*spectypes.Share,
	qbftController *controller.Controller,
	beacon specssv.BeaconNode,
	network specssv.Network,
	signer spectypes.KeyManager,
	valCheck specqbft.ProposedValueCheckF,
	highestDecidedSlot phase0.Slot,
) Runner {
	return &SyncCommitteeAggregatorRunner{
		BaseRunner: &BaseRunner{
			BeaconRoleType:     spectypes.BNRoleSyncCommitteeContribution,
			BeaconNetwork:      beaconNetwork,
			Shares:             *shares,
			QBFTController:     qbftController,
			highestDecidedSlot: highestDecidedSlot,
		},

		beacon:   beacon,
		network:  network,
		signer:   signer,
		valCheck: valCheck,
		metrics:  metrics.NewConsensusMetrics(spectypes.BNRoleSyncCommitteeContribution),
	}
}

func (r *SyncCommitteeAggregatorRunner) StartNewDuty(logger *zap.Logger, duty spectypes.Duty) error {
	return r.BaseRunner.baseStartNewDuty(logger, r, duty)
}

// HasRunningDuty returns true if a duty is already running (StartNewDuty called and returned nil)
func (r *SyncCommitteeAggregatorRunner) HasRunningDuty() bool {
	return r.BaseRunner.hasRunningDuty()
}

func (r *SyncCommitteeAggregatorRunner) ProcessPreConsensus(logger *zap.Logger, signedMsg ssvtypes.PartialSignatureMessages) error {
	quorum, roots, err := r.BaseRunner.basePreConsensusMsgProcessing(r, signedMsg)
	if err != nil {
		return errors.Wrap(err, "failed processing sync committee selection proof message")
	}

	// quorum returns true only once (first time quorum achieved)
	if !quorum {
		return nil
	}

	r.metrics.EndPreConsensus()

	// collect selection proofs and subnets
	var (
		selectionProofs []phase0.BLSSignature
		subnets         []uint64
	)
	for i, root := range roots {
		// reconstruct selection proof sig
		sig, err := r.GetState().ReconstructBeaconSig(r.GetState().PreConsensusContainer, root, r.GetShare().ValidatorPubKey[:], r.GetShare().ValidatorIndex)
		if err != nil {
			// If the reconstructed signature verification failed, fall back to verifying each partial signature
			for _, root := range roots {
				r.BaseRunner.FallBackAndVerifyEachSignature(r.GetState().PreConsensusContainer, root, r.GetShare().Committee, r.GetShare().ValidatorIndex)
			}
			return errors.Wrap(err, "got pre-consensus quorum but it has invalid signatures")
		}
		blsSigSelectionProof := phase0.BLSSignature{}
		copy(blsSigSelectionProof[:], sig)

		aggregator, err := r.GetBeaconNode().IsSyncCommitteeAggregator(sig)
		if err != nil {
			return errors.Wrap(err, "could not check if sync committee aggregator")
		}
		if !aggregator {
			continue
		}

		// fetch sync committee contribution
		subnet, err := r.GetBeaconNode().SyncCommitteeSubnetID(phase0.CommitteeIndex(r.GetState().StartingDuty.ValidatorSyncCommitteeIndices[i]))
		if err != nil {
			return errors.Wrap(err, "could not get sync committee subnet ID")
		}

		selectionProofs = append(selectionProofs, blsSigSelectionProof)
		subnets = append(subnets, subnet)
	}
	if len(selectionProofs) == 0 {
		r.GetState().Finished = true
		return nil
	}

	duty := r.GetState().StartingDuty

	// fetch contributions
	r.metrics.PauseDutyFullFlow()
	contributions, ver, err := r.GetBeaconNode().GetSyncCommitteeContribution(duty.DutySlot(), selectionProofs, subnets)
	if err != nil {
		return errors.Wrap(err, "could not get sync committee contribution")
	}
	r.metrics.ContinueDutyFullFlow()

	byts, err := contributions.MarshalSSZ()
	if err != nil {
		return errors.Wrap(err, "could not marshal contributions")
	}

	// create consensus object
	input := &spectypes.ConsensusData{
		Duty:    *duty,
		Version: ver,
		DataSSZ: byts,
	}

	r.metrics.StartConsensus()
	if err := r.BaseRunner.decide(logger, r, input); err != nil {
		return errors.Wrap(err, "can't start new duty runner instance for duty")
	}
	return nil
}

func (r *SyncCommitteeAggregatorRunner) ProcessConsensus(logger *zap.Logger, signedMsg ssvtypes.SignedMessage) error {
	decided, decidedValue, err := r.BaseRunner.baseConsensusMsgProcessing(logger, r, signedMsg)
	if err != nil {
		return errors.Wrap(err, "failed processing consensus message")
	}

	// Decided returns true only once so if it is true it must be for the current running instance
	if !decided {
		return nil
	}

	r.metrics.EndConsensus()
	r.metrics.StartPostConsensus()

	contributions, err := decidedValue.GetSyncCommitteeContributions()
	if err != nil {
		return errors.Wrap(err, "could not get contributions")
	}

	// specific duty sig
	msgs := make([]*spectypes.PartialSignatureMessage, 0)
	for _, c := range contributions {
		contribAndProof, _, err := r.generateContributionAndProof(c.Contribution, c.SelectionProofSig)
		if err != nil {
			return errors.Wrap(err, "could not generate contribution and proof")
		}

		signed, err := r.BaseRunner.signBeaconObject(r, r.BaseRunner.State.StartingDuty.(*spectypes.BeaconDuty), contribAndProof,
			decidedValue.duty.DutySlot(),
			spectypes.DomainContributionAndProof)
		if err != nil {
			return errors.Wrap(err, "failed to sign aggregate and proof")
		}

		msgs = append(msgs, signed)
	}
	postConsensusMsg := &spectypes.PartialSignatureMessages{
		Type:     spectypes.PostConsensusPartialSig,
		Slot:     decidedValue.duty.DutySlot(),
		Messages: msgs,
	}

	postSignedMsg, err := r.BaseRunner.signPostConsensusMsg(r, postConsensusMsg)
	if err != nil {
		return errors.Wrap(err, "could not sign post consensus msg")
	}

	data, err := postSignedMsg.Encode()
	if err != nil {
		return errors.Wrap(err, "failed to encode post consensus signature msg")
	}

	msgToBroadcast := &genesisspectypes.SSVMessage{
		MsgType: genesisspectypes.SSVPartialSignatureMsgType,
		MsgID:   genesisspectypes.NewMsgID(genesisspectypes.DomainType(r.GetShare().DomainType), r.GetShare().ValidatorPubKey[:], r.BaseRunner.BeaconRoleType),
		Data:    data,
	}

	if err := r.GetNetwork().Broadcast(msgToBroadcast); err != nil {
		return errors.Wrap(err, "can't broadcast partial post consensus sig")
	}
	return nil
}

func (r *SyncCommitteeAggregatorRunner) ProcessPostConsensus(logger *zap.Logger, signedMsg ssvtypes.PartialSignatureMessages) error {
	quorum, roots, err := r.BaseRunner.basePostConsensusMsgProcessing(logger, r, signedMsg)
	if err != nil {
		return errors.Wrap(err, "failed processing post consensus message")
	}

	if !quorum {
		return nil
	}

	r.metrics.EndPostConsensus()

	// get contributions
	contributions, err := r.GetState().DecidedValue.GetSyncCommitteeContributions()
	if err != nil {
		return errors.Wrap(err, "could not get contributions")
	}

	for _, root := range roots {
		sig, err := r.GetState().ReconstructBeaconSig(r.GetState().PostConsensusContainer, root, r.GetShare().ValidatorPubKey[:], r.GetShare().ValidatorIndex)
		if err != nil {
			// If the reconstructed signature verification failed, fall back to verifying each partial signature
			for _, root := range roots {
				r.BaseRunner.FallBackAndVerifyEachSignature(r.GetState().PostConsensusContainer, root, r.GetShare().Committee, r.GetShare().ValidatorIndex)
			}
			return errors.Wrap(err, "got post-consensus quorum but it has invalid signatures")
		}
		specSig := phase0.BLSSignature{}
		copy(specSig[:], sig)

		for _, contribution := range contributions {
			// match the right contrib and proof root to signed root
			contribAndProof, contribAndProofRoot, err := r.generateContributionAndProof(contribution.Contribution, contribution.SelectionProofSig)
			if err != nil {
				return errors.Wrap(err, "could not generate contribution and proof")
			}
			if !bytes.Equal(root[:], contribAndProofRoot[:]) {
				continue // not the correct root
			}

			signedContrib, err := r.GetState().ReconstructBeaconSig(r.GetState().PostConsensusContainer, root, r.GetShare().ValidatorPubKey[:], r.GetShare().ValidatorIndex)
			if err != nil {
				return errors.Wrap(err, "could not reconstruct contribution and proof sig")
			}
			blsSignedContribAndProof := phase0.BLSSignature{}
			copy(blsSignedContribAndProof[:], signedContrib)
			signedContribAndProof := &altair.SignedContributionAndProof{
				Message:   contribAndProof,
				Signature: blsSignedContribAndProof,
			}

			submissionEnd := r.metrics.StartBeaconSubmission()

			if err := r.GetBeaconNode().SubmitSignedContributionAndProof(signedContribAndProof); err != nil {
				r.metrics.RoleSubmissionFailed()
				return errors.Wrap(err, "could not submit to Beacon chain reconstructed contribution and proof")
			}

			submissionEnd()
			r.metrics.EndDutyFullFlow(r.GetState().RunningInstance.State.Round)
			r.metrics.RoleSubmitted()

			logger.Debug("✅ submitted successfully sync committee aggregator!")
			break
		}
	}
	r.GetState().Finished = true
	return nil
}

func (r *SyncCommitteeAggregatorRunner) generateContributionAndProof(contrib altair.SyncCommitteeContribution, proof phase0.BLSSignature) (*altair.ContributionAndProof, phase0.Root, error) {
	contribAndProof := &altair.ContributionAndProof{
		AggregatorIndex: r.GetState().DecidedValue.Duty.ValidatorIndex,
		Contribution:    &contrib,
		SelectionProof:  proof,
	}

	epoch := r.BaseRunner.BeaconNetwork.EstimatedEpochAtSlot(r.GetState().DecidedValue.duty.DutySlot())
	dContribAndProof, err := r.GetBeaconNode().DomainData(epoch, spectypes.DomainContributionAndProof)
	if err != nil {
		return nil, phase0.Root{}, errors.Wrap(err, "could not get domain data")
	}
	contribAndProofRoot, err := spectypes.ComputeETHSigningRoot(contribAndProof, dContribAndProof)
	if err != nil {
		return nil, phase0.Root{}, errors.Wrap(err, "could not compute signing root")
	}
	return contribAndProof, contribAndProofRoot, nil
}

func (r *SyncCommitteeAggregatorRunner) expectedPreConsensusRootsAndDomain() ([]ssz.HashRoot, phase0.DomainType, error) {
	sszIndexes := make([]ssz.HashRoot, 0)
	for _, index := range r.GetState().StartingDuty.ValidatorSyncCommitteeIndices {
		subnet, err := r.GetBeaconNode().SyncCommitteeSubnetID(phase0.CommitteeIndex(index))
		if err != nil {
			return nil, spectypes.DomainError, errors.Wrap(err, "could not get sync committee subnet ID")
		}
		data := &altair.SyncAggregatorSelectionData{
			Slot:              r.GetState().StartingDuty.Slot,
			SubcommitteeIndex: subnet,
		}
		sszIndexes = append(sszIndexes, data)
	}
	return sszIndexes, spectypes.DomainSyncCommitteeSelectionProof, nil
}

// expectedPostConsensusRootsAndDomain an INTERNAL function, returns the expected post-consensus roots to sign
func (r *SyncCommitteeAggregatorRunner) expectedPostConsensusRootsAndDomain() ([]ssz.HashRoot, phase0.DomainType, error) {
	// get contributions
	contributions, err := r.GetState().DecidedValue.GetSyncCommitteeContributions()
	if err != nil {
		return nil, phase0.DomainType{}, errors.Wrap(err, "could not get contributions")
	}

	ret := make([]ssz.HashRoot, 0)
	for _, contrib := range contributions {
		contribAndProof, _, err := r.generateContributionAndProof(contrib.Contribution, contrib.SelectionProofSig)
		if err != nil {
			return nil, spectypes.DomainError, errors.Wrap(err, "could not generate contribution and proof")
		}
		ret = append(ret, contribAndProof)
	}
	return ret, spectypes.DomainContributionAndProof, nil
}

// executeDuty steps:
// 1) sign a partial contribution proof (for each subcommittee index) and wait for 2f+1 partial sigs from peers
// 2) Reconstruct contribution proofs, check IsSyncCommitteeAggregator and start consensus on duty + contribution data
// 3) Once consensus decides, sign partial contribution data (for each subcommittee) and broadcast
// 4) collect 2f+1 partial sigs, reconstruct and broadcast valid SignedContributionAndProof (for each subcommittee) sig to the BN
func (r *SyncCommitteeAggregatorRunner) executeDuty(logger *zap.Logger, duty spectypes.Duty) error {
	r.metrics.StartDutyFullFlow()
	r.metrics.StartPreConsensus()

	// sign selection proofs
	msgs := genesisspectypes.PartialSignatureMessages{
		Type:     genesisspectypes.ContributionProofs,
		Slot:     duty.DutySlot(),
		Messages: []*genesisspectypes.PartialSignatureMessage{},
	}
	for _, index := range r.GetState().StartingDuty.ValidatorSyncCommitteeIndices {
		subnet, err := r.GetBeaconNode().SyncCommitteeSubnetID(phase0.CommitteeIndex(index))
		if err != nil {
			return errors.Wrap(err, "could not get sync committee subnet ID")
		}
		data := &altair.SyncAggregatorSelectionData{
			Slot:              duty.DutySlot(),
			SubcommitteeIndex: subnet,
		}
		msg, err := r.BaseRunner.signBeaconObject(r, duty.(*spectypes.BeaconDuty), data, duty.DutySlot(),
			spectypes.DomainSyncCommitteeSelectionProof)
		if err != nil {
			return errors.Wrap(err, "could not sign sync committee selection proof")
		}

		msgs.Messages = append(msgs.Messages, msg)
	}

	// package into signed partial sig
	signature, err := r.GetGenesisSigner().SignRoot(msgs, genesisspectypes.PartialSignatureType, r.GetShare().SharePubKey)
	if err != nil {
		return errors.Wrap(err, "could not sign PartialSignatureMessage for contribution proofs")
	}
	signedPartialMsg := &genesisspectypes.SignedPartialSignatureMessage{
		Message:   msgs,
		Signature: signature,
		Signer:    r.GetOperatorSigner().GetOperatorID(),
	}

	// broadcast
	data, err := signedPartialMsg.Encode()
	if err != nil {
		return errors.Wrap(err, "failed to encode contribution proofs pre-consensus signature msg")
	}
	msgToBroadcast := &genesisspectypes.SSVMessage{
		MsgType: genesisspectypes.SSVPartialSignatureMsgType,
		MsgID:   genesisspectypes.NewMsgID(genesisspectypes.DomainType(r.GetShare().DomainType), r.GetShare().ValidatorPubKey[:], r.BaseRunner.BeaconRoleType),
		Data:    data,
	}
	if err := r.GetNetwork().Broadcast(msgToBroadcast); err != nil {
		return errors.Wrap(err, "can't broadcast partial contribution proof sig")
	}
	return nil
}

func (r *SyncCommitteeAggregatorRunner) GetBaseRunner() *BaseRunner {
	return r.BaseRunner
}

func (r *SyncCommitteeAggregatorRunner) GetNetwork() specssv.Network {
	return r.network
}

func (r *SyncCommitteeAggregatorRunner) GetBeaconNode() specssv.BeaconNode {
	return r.beacon
}

func (r *SyncCommitteeAggregatorRunner) GetShare() *spectypes.Share {
	for _, share := range r.BaseRunner.Shares {
		return share
	}
	return nil
}

func (r *SyncCommitteeAggregatorRunner) GetState() *State {
	return r.BaseRunner.State
}

func (r *SyncCommitteeAggregatorRunner) GetValCheckF() specqbft.ProposedValueCheckF {
	return r.valCheck
}

func (r *SyncCommitteeAggregatorRunner) GetGenesisSigner() genesisspectypes.KeyManager {
	return r.signer
}

func (r *SyncCommitteeAggregatorRunner) GetSigner() spectypes.BeaconSigner {
	return r.beaconSigner
}

func (r *SyncCommitteeAggregatorRunner) GetOperatorSigner() spectypes.OperatorSigner {
	return r.operatorSigner
}

// Encode returns the encoded struct in bytes or error
func (r *SyncCommitteeAggregatorRunner) Encode() ([]byte, error) {
	return json.Marshal(r)
}

// Decode returns error if decoding failed
func (r *SyncCommitteeAggregatorRunner) Decode(data []byte) error {
	return json.Unmarshal(data, &r)
}

// GetRoot returns the root used for signing and verification
func (r *SyncCommitteeAggregatorRunner) GetRoot() ([32]byte, error) {
	marshaledRoot, err := r.Encode()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not encode DutyRunnerState")
	}
	ret := sha256.Sum256(marshaledRoot)
	return ret, nil
}
