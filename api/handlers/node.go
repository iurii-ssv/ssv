package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/bloxapp/ssv/api"
	networkpeers "github.com/bloxapp/ssv/network/peers"
	"github.com/bloxapp/ssv/nodeprobe"
)

type TopicIndex interface {
	PeersByTopic() ([]peer.ID, map[string][]peer.ID)
}

type HealthStatus int

const (
	NotHealthy HealthStatus = iota
	Healthy
)

func (c HealthStatus) String() string {
	str := [...]string{"NotHealthy", "Healthy"}
	if c < 0 || int(c) >= len(str) {
		return "(unrecognized)"
	}
	return str[c]
}

type AllPeersAndTopicsJSON struct {
	AllPeers     []peer.ID        `json:"all_peers"`
	PeersByTopic []topicIndexJSON `json:"peers_by_topic"`
}

type topicIndexJSON struct {
	TopicName string    `json:"topic"`
	Peers     []peer.ID `json:"peers"`
}

type connectionJSON struct {
	Address   string `json:"address"`
	Direction string `json:"direction"`
}

type peerJSON struct {
	ID            peer.ID          `json:"id"`
	Addresses     []string         `json:"addresses"`
	Connections   []connectionJSON `json:"connections"`
	Connectedness string           `json:"connectedness"`
	Subnets       string           `json:"subnets"`
	Version       string           `json:"version"`
}

type identityJSON struct {
	PeerID    peer.ID  `json:"peer_id"`
	Addresses []string `json:"addresses"`
	Subnets   string   `json:"subnets"`
	Version   string   `json:"version"`
}

type healthCheckJSON struct {
	PeersHealthStatus               string   `json:"peers_status"`
	BeaconConnectionHealthStatus    string   `json:"beacon_health_status"`
	ExecutionConnectionHealthStatus string   `json:"execution_health_status"`
	EventSyncHealthStatus           string   `json:"event_sync_health_status"`
	LocalPortsListening             []string `json:"local_ports_listening"`
}

type Node struct {
	PeersIndex networkpeers.Index
	TopicIndex TopicIndex
	Network    network.Network
	NodeProber *nodeprobe.Prober
}

func (h *Node) Identity(w http.ResponseWriter, r *http.Request) error {
	nodeInfo := h.PeersIndex.Self()
	resp := identityJSON{
		PeerID:  h.Network.LocalPeer(),
		Subnets: nodeInfo.Metadata.Subnets,
		Version: nodeInfo.Metadata.NodeVersion,
	}
	for _, addr := range h.Network.ListenAddresses() {
		resp.Addresses = append(resp.Addresses, addr.String())
	}
	return api.Render(w, r, resp)
}

func (h *Node) Peers(w http.ResponseWriter, r *http.Request) error {
	prs := h.Network.Peers()
	resp := h.peers(prs)
	return api.Render(w, r, resp)
}

func (h *Node) Topics(w http.ResponseWriter, r *http.Request) error {
	allpeers, peerbytpc := h.TopicIndex.PeersByTopic()
	alland := AllPeersAndTopicsJSON{}
	tpcs := []topicIndexJSON{}
	for topic, peerz := range peerbytpc {
		tpcs = append(tpcs, topicIndexJSON{TopicName: topic, Peers: peerz})
	}
	alland.AllPeers = allpeers
	alland.PeersByTopic = tpcs

	return api.Render(w, r, alland)
}

func (h *Node) Health(w http.ResponseWriter, r *http.Request) error {
	ctx := context.Background()
	resp := healthCheckJSON{}
	// check ports being used
	addrs := h.Network.ListenAddresses()
	for _, addr := range addrs {
		resp.LocalPortsListening = append(resp.LocalPortsListening, addr.String())
	}
	// check consensus node health
	err := h.NodeProber.CheckBeaconNodeHealth(ctx)
	if err == nil {
		resp.BeaconConnectionHealthStatus = Healthy.String()
	} else {
		resp.BeaconConnectionHealthStatus = fmt.Sprintf("%s: %s", NotHealthy.String(), err.Error())
	}
	// check execution node health
	err = h.NodeProber.CheckExecutionNodeHealth(ctx)
	if err == nil {
		resp.ExecutionConnectionHealthStatus = Healthy.String()
	} else {
		resp.ExecutionConnectionHealthStatus = fmt.Sprintf("%s: %s", NotHealthy.String(), err.Error())
	}
	// check event sync health
	err = h.NodeProber.CheckEventSycNodeHealth(ctx)
	if err != nil {
		resp.EventSyncHealthStatus = Healthy.String()
	} else {
		resp.EventSyncHealthStatus = fmt.Sprintf("%s: %s", NotHealthy.String(), err.Error())
	}
	// check peers connection
	var activePeerCount int
	prs := h.Network.Peers()
	for _, p := range h.peers(prs) {
		if p.Connectedness == "Connected" {
			activePeerCount++
		}
	}
	switch count := activePeerCount; {
	case count >= 5:
		resp.PeersHealthStatus = fmt.Sprintf("%s: %d  peers are connected", Healthy.String(), activePeerCount)
	case count < 5:
		resp.PeersHealthStatus = fmt.Sprintf("%s: %s", NotHealthy.String(), "less than 5 peers are connected")
	case count == 0:
		resp.PeersHealthStatus = fmt.Sprintf("%s: %s", NotHealthy.String(), "error: no peers are connected")
	}
	return api.Render(w, r, resp)
}

func (h *Node) peers(peers []peer.ID) []peerJSON {
	resp := make([]peerJSON, len(peers))
	for i, id := range peers {
		resp[i] = peerJSON{
			ID:            id,
			Connectedness: h.Network.Connectedness(id).String(),
			Subnets:       h.PeersIndex.GetPeerSubnets(id).String(),
		}

		for _, addr := range h.Network.Peerstore().Addrs(id) {
			resp[i].Addresses = append(resp[i].Addresses, addr.String())
		}

		conns := h.Network.ConnsToPeer(id)
		for _, conn := range conns {
			resp[i].Connections = append(resp[i].Connections, connectionJSON{
				Address:   conn.RemoteMultiaddr().String(),
				Direction: conn.Stat().Direction.String(),
			})
		}

		nodeInfo := h.PeersIndex.NodeInfo(id)
		if nodeInfo == nil {
			continue
		}
		resp[i].Version = nodeInfo.Metadata.NodeVersion
	}
	return resp
}
