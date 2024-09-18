package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	spectypes "github.com/ssvlabs/ssv-spec/types"
	"github.com/ssvlabs/ssv/network/peers"
	"github.com/ssvlabs/ssv/network/records"
	"github.com/ssvlabs/ssv/networkconfig"
)

var (
	testIP      = "127.0.0.1"
	testBindIP  = "127.0.0.1"
	testPort    = 12001
	testTCPPort = 13001
)

func createServiceOptions(t *testing.T, networkConfig networkconfig.NetworkConfig) *Options {
	// Generate key
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	// Discv5 options
	discV5Opts := &DiscV5Options{
		StoragePath: t.TempDir(),
		IP:          testIP,
		BindIP:      testBindIP,

		Port:          testPort,
		TCPPort:       testTCPPort,
		NetworkKey:    privKey,
		Bootnodes:     networkConfig.Bootnodes,
		Subnets:       mockSubnets(1),
		EnableLogging: false,
	}

	// Service options
	allSubs, _ := records.Subnets{}.FromString(records.AllSubnets)
	subnetsIndex := peers.NewSubnetsIndex(len(allSubs))
	connectionIndex := NewMockConnection()

	return &Options{
		DiscV5Opts:    discV5Opts,
		ConnIndex:     connectionIndex,
		SubnetsIdx:    subnetsIndex,
		NetworkConfig: networkConfig,
	}
}

func testingService(t *testing.T) *DiscV5Service {
	opts := createServiceOptions(t, testNetConfig)
	service, err := newDiscV5Service(testCtx, testLogger, opts)
	require.NoError(t, err)
	require.NotNil(t, service)

	dvs, ok := service.(*DiscV5Service)
	require.True(t, ok)

	return dvs
}

func TestNewDiscV5Service(t *testing.T) {
	dvs := testingService(t)

	assert.NotNil(t, dvs.dv5Listener)
	assert.NotNil(t, dvs.conns)
	assert.NotNil(t, dvs.subnetsIdx)
	assert.NotNil(t, dvs.networkConfig)

	// Check bootnodes
	for _, bootnode := range testNetConfig.Bootnodes {
		nodes, err := ParseENR(nil, false, bootnode)
		require.NoError(t, err)
		require.Contains(t, dvs.bootnodes, nodes[0])
	}

	// Close
	err := dvs.Close()
	require.NoError(t, err)
}

func TestDiscV5Service_Close(t *testing.T) {
	dvs := testingService(t)

	err := dvs.Close()
	assert.NoError(t, err)
}

func TestDiscV5Service_RegisterSubnets(t *testing.T) {
	dvs := testingService(t)

	// Register subnets 1, 3, and 5
	updated, err := dvs.RegisterSubnets(testLogger, 1, 3, 5)
	assert.NoError(t, err)
	assert.True(t, updated)

	require.Equal(t, byte(1), dvs.subnets[1])
	require.Equal(t, byte(1), dvs.subnets[3])
	require.Equal(t, byte(1), dvs.subnets[5])
	require.Equal(t, byte(0), dvs.subnets[2])

	// Register the same subnets. Should not update the state
	updated, err = dvs.RegisterSubnets(testLogger, 1, 3, 5)
	assert.NoError(t, err)
	assert.False(t, updated)

	require.Equal(t, byte(1), dvs.subnets[1])
	require.Equal(t, byte(1), dvs.subnets[3])
	require.Equal(t, byte(1), dvs.subnets[5])
	require.Equal(t, byte(0), dvs.subnets[2])

	// Register different subnets
	updated, err = dvs.RegisterSubnets(testLogger, 2, 4)
	assert.NoError(t, err)
	assert.True(t, updated)
	require.Equal(t, byte(1), dvs.subnets[1])
	require.Equal(t, byte(1), dvs.subnets[2])
	require.Equal(t, byte(1), dvs.subnets[3])
	require.Equal(t, byte(1), dvs.subnets[4])
	require.Equal(t, byte(1), dvs.subnets[5])
	require.Equal(t, byte(0), dvs.subnets[6])

	// Close
	err = dvs.Close()
	require.NoError(t, err)
}

func TestDiscV5Service_DeregisterSubnets(t *testing.T) {
	dvs := testingService(t)

	// Register subnets first
	_, err := dvs.RegisterSubnets(testLogger, 1, 2, 3)
	require.NoError(t, err)

	require.Equal(t, byte(1), dvs.subnets[1])
	require.Equal(t, byte(1), dvs.subnets[2])
	require.Equal(t, byte(1), dvs.subnets[3])

	// Deregister from 2 and 3
	updated, err := dvs.DeregisterSubnets(testLogger, 2, 3)
	assert.NoError(t, err)
	assert.True(t, updated)

	require.Equal(t, byte(1), dvs.subnets[1])
	require.Equal(t, byte(0), dvs.subnets[2])
	require.Equal(t, byte(0), dvs.subnets[3])

	// Deregistering non-existent subnets should not update
	updated, err = dvs.DeregisterSubnets(testLogger, 4, 5)
	assert.NoError(t, err)
	assert.False(t, updated)

	// Close
	err = dvs.Close()
	require.NoError(t, err)
}

func TestDiscV5Service_PublishENR(t *testing.T) {
	logger := zap.NewNop()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	opts := createServiceOptions(t, testNetConfig)
	service, err := newDiscV5Service(ctx, testLogger, opts)
	require.NoError(t, err)
	dvs := service.(*DiscV5Service)

	// Replace listener
	err = dvs.conn.Close()
	require.NoError(t, err)
	dvs.dv5Listener = NewMockListener(dvs.Self(), []*enode.Node{NewTestingNode(t)})

	// Test PublishENR method
	dvs.PublishENR(logger)

	// Verify that the publish state is reset to ready
	assert.Eventually(t, func() bool {
		return dvs.publishState == publishStateReady
	}, time.Second, 10*time.Millisecond)
}

func TestDiscV5Service_Bootstrap(t *testing.T) {
	logger := zap.NewNop()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	opts := createServiceOptions(t, testNetConfig)

	service, err := newDiscV5Service(testCtx, testLogger, opts)
	require.NoError(t, err)

	dvs := service.(*DiscV5Service)

	// Replace listener
	err = dvs.conn.Close()
	require.NoError(t, err)
	testingNode := NewTestingNode(t)
	dvs.dv5Listener = NewMockListener(dvs.Self(), []*enode.Node{testingNode})

	// testing handler. It's called whenever a new peer is found
	handlerCalled := make(chan struct{})
	handler := func(e PeerEvent) {
		require.Equal(t, testingNode, e.Node)
		close(handlerCalled)
	}

	// Run bootstrap
	go func() {
		err := dvs.Bootstrap(logger, handler)
		assert.NoError(t, err)
	}()

	// Wait for testing peer to be found
	select {
	case <-handlerCalled:
		// Test passed
	case <-ctx.Done():
		t.Fatal("Bootstrap timed out")
	}
}

func TestDiscV5Service_Node(t *testing.T) {
	dvs := testingService(t)

	// Replace listener
	err := dvs.conn.Close()
	require.NoError(t, err)
	testingNode := NewTestingNode(t)
	dvs.dv5Listener = NewMockListener(dvs.Self(), []*enode.Node{testingNode})

	// Create a mock peer.AddrInfo
	unknownPeer := NewTestingNode(t)
	unknownPeerAddrInfo, err := ToPeer(unknownPeer)
	assert.NoError(t, err)

	// Test looking for an unknown peer
	node, err := dvs.Node(testLogger, *unknownPeerAddrInfo)
	assert.NoError(t, err)
	assert.Nil(t, node)

	// Test looking for a known peer
	addrInfo, err := ToPeer(testingNode)
	assert.NoError(t, err)
	node, err = dvs.Node(testLogger, *addrInfo)
	assert.NoError(t, err)
	assert.Equal(t, testingNode, node)
}

func TestDiscV5Service_checkPeer(t *testing.T) {
	dvs := testingService(t)

	// Valid peer
	err := dvs.checkPeer(testLogger, ToPeerEvent(NewTestingNode(t)))
	require.NoError(t, err)

	// No domain
	err = dvs.checkPeer(testLogger, ToPeerEvent(NodeWithoutDomain(t)))
	require.ErrorContains(t, err, "could not read domain type: not found")

	// No next domain. No error since it's not enforced
	err = dvs.checkPeer(testLogger, ToPeerEvent(NodeWithoutNextDomain(t)))
	require.NoError(t, err)

	// Matching main domain
	err = dvs.checkPeer(testLogger, ToPeerEvent(NodeWithCustomDomains(t, testNetConfig.DomainType(), spectypes.DomainType{})))
	require.NoError(t, err)

	// Matching next domain
	err = dvs.checkPeer(testLogger, ToPeerEvent(NodeWithCustomDomains(t, spectypes.DomainType{}, testNetConfig.DomainType())))
	require.NoError(t, err)

	// Mismatching domains
	err = dvs.checkPeer(testLogger, ToPeerEvent(NodeWithCustomDomains(t, spectypes.DomainType{}, spectypes.DomainType{})))
	require.ErrorContains(t, err, "mismatched domain type: neither 00000000 nor 00000000 match 00000502")

	// No subnets
	err = dvs.checkPeer(testLogger, ToPeerEvent(NodeWithoutSubnets(t)))
	require.ErrorContains(t, err, "could not read subnets: not found")

	// Zero subnets
	err = dvs.checkPeer(testLogger, ToPeerEvent(NodeWithZeroSubnets(t)))
	require.ErrorContains(t, err, "zero subnets")

	// Valid peer but reached limit
	dvs.conns.(*MockConnection).SetAtLimit(true)
	err = dvs.checkPeer(testLogger, ToPeerEvent(NewTestingNode(t)))
	require.ErrorContains(t, err, "reached limit")
	dvs.conns.(*MockConnection).SetAtLimit(false)

	// Valid peer but no common subnet
	subnets := make([]byte, len(records.ZeroSubnets))
	subnets[10] = 1
	err = dvs.checkPeer(testLogger, ToPeerEvent(NodeWithCustomSubnets(t, subnets)))
	require.ErrorContains(t, err, "no shared subnets")
}