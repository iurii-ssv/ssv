package storage

import (
	"testing"

	qbftstorage "github.com/bloxapp/ssv/protocol/v2/qbft/storage"
	"github.com/bloxapp/ssv/storage/basedb"
	"github.com/bloxapp/ssv/storage/kv"
	specqbft "github.com/ssvlabs/ssv-spec-pre-cc/qbft"

	"github.com/ssvlabs/ssv-spec-pre-cc/types"
	"github.com/stretchr/testify/require"

	"github.com/bloxapp/ssv/logging"
)

func TestQBFTStores(t *testing.T) {
	logger := logging.TestLogger(t)

	qbftMap := NewStores()

	store, err := newTestIbftStorage(logger, "")
	require.NoError(t, err)
	qbftMap.Add(types.BNRoleAttester, store)
	qbftMap.Add(types.BNRoleProposer, store)

	require.NotNil(t, qbftMap.Get(types.BNRoleAttester))
	require.NotNil(t, qbftMap.Get(types.BNRoleProposer))

	db, err := kv.NewInMemory(logger.Named(logging.NameBadgerDBLog), basedb.Options{
		Reporting: true,
	})
	require.NoError(t, err)
	qbftMap = NewStoresFromRoles(db, types.BNRoleAttester, types.BNRoleProposer)

	require.NotNil(t, qbftMap.Get(types.BNRoleAttester))
	require.NotNil(t, qbftMap.Get(types.BNRoleProposer))

	id := []byte{1, 2, 3}

	qbftMap.Each(func(role types.BeaconRole, store qbftstorage.QBFTStore) error {
		return store.SaveInstance(&qbftstorage.StoredInstance{State: &specqbft.State{Height: 1, ID: id}})
	})

	instance, err := qbftMap.Get(types.BNRoleAttester).GetInstance(id, 1)
	require.NoError(t, err)
	require.NotNil(t, instance)
	require.Equal(t, specqbft.Height(1), instance.State.Height)
	require.Equal(t, id, instance.State.ID)
}
