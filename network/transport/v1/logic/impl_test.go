/*
 * Copyright (C) 2021 Nuts community
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/nuts-foundation/nuts-node/crypto/hash"
	"github.com/nuts-foundation/nuts-node/network/dag"
	"github.com/nuts-foundation/nuts-node/network/transport"
	"github.com/nuts-foundation/nuts-node/network/transport/grpc"
	"github.com/stretchr/testify/assert"
)

func Test_ProtocolLifecycle(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	state := dag.NewMockState(mockCtrl)
	state.EXPECT().Subscribe(dag.TransactionAddedEvent, dag.AnyPayloadType, gomock.Any())

	instance := NewProtocol(NewMockMessageGateway(mockCtrl), nil, state, nil)
	instance.Configure(time.Second*2, time.Second*5, 10*time.Second, "local")
	instance.Start()
	instance.Stop()
}

func Test_Protocol_PeerDiagnostics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	connection := grpc.NewMockConnection(ctrl)
	connection.EXPECT().IsConnected().Return(true)
	connectionList := grpc.NewMockConnectionList(ctrl)
	connectionList.EXPECT().Get(grpc.ByPeerID(peerID)).Return(connection)
	connectionList.EXPECT().Get(grpc.ByPeerID("disconnected-peer")).Return(nil)

	instance := NewProtocol(nil, nil, nil, nil).(*protocol)
	instance.connections = connectionList

	instance.peerDiagnostics[peerID] = transport.Diagnostics{
		Peers:           []transport.PeerID{peerID},
		SoftwareVersion: "1.0",
	}

	// Add a "disconnected" peer to test clean up
	instance.peerDiagnostics["disconnected-peer"] = transport.Diagnostics{}

	diagnostics := instance.PeerDiagnostics()
	instance.peerDiagnostics[peerID].Peers[0] = "other-peer" // mutate entry to make sure function returns a copy
	assert.Len(t, diagnostics, 1)
	actual := diagnostics[peerID]
	assert.Equal(t, "1.0", actual.SoftwareVersion)
	assert.Equal(t, []transport.PeerID{peerID}, actual.Peers)
}

func Test_Protocol_StartAdvertingDiagnostics(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		instance := NewProtocol(nil, nil, nil, nil).(*protocol)
		instance.advertDiagnosticsInterval = 0 * time.Second // this is what would be configured
		instance.startAdvertingDiagnostics(nil)
		// This is a blocking function when the feature is enabled, so if we reach the end of the test everything works as intended.
	})
}

func Test_Protocol_StartUpdatingDiagnostics(t *testing.T) {
	t.Run("context cancel", func(t *testing.T) {
		instance := NewProtocol(nil, nil, nil, nil).(*protocol)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		instance.startUpdatingDiagnostics(ctx) // should exit immediately
	})
}

func Test_Protocol_Diagnostics(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		connection := grpc.NewMockConnection(ctrl)
		connection.EXPECT().IsConnected().Return(true)
		connectionList := grpc.NewMockConnectionList(ctrl)
		connectionList.EXPECT().Get(grpc.ByPeerID(peerID)).Return(connection)
		connectionList.EXPECT().Get(grpc.ByPeerID("disconnected-peer")).Return(nil)

		payloadCollector := NewMockmissingPayloadCollector(ctrl)
		payloadCollector.EXPECT().findMissingPayloads().AnyTimes().Return(nil, nil)

		instance := NewProtocol(nil, nil, nil, nil).(*protocol)
		instance.connections = connectionList
		instance.missingPayloadCollector = payloadCollector
		instance.peerOmnihashChannel = make(chan PeerOmnihash, 1)

		// Add something to clean up
		instance.peerOmnihashes["disconnected-peer"] = hash.SHA256Hash{1}

		stats := instance.Diagnostics()[0].(peerOmnihashStatistic)
		assert.Empty(t, stats.peerHashes)

		// Peer broadcasts hash
		peerHash := hash.SHA256Sum([]byte("Hello, World!"))
		instance.peerOmnihashChannel <- PeerOmnihash{Peer: peerID, Hash: peerHash}
		instance.updateDiagnostics(nil)
		stats = instance.Diagnostics()[0].(peerOmnihashStatistic)
		assert.Len(t, stats.peerHashes, 1)
		assert.Equal(t, peerHash, stats.peerHashes[peerID])
	})

	t.Run("ok - missing payloads", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		payloadCollector := NewMockmissingPayloadCollector(ctrl)
		payloadCollector.EXPECT().findMissingPayloads().Return([]hash.SHA256Hash{{1}}, nil)

		instance := NewProtocol(nil, nil, nil, nil).(*protocol)
		instance.missingPayloadCollector = payloadCollector
		diagnostics := instance.Diagnostics()
		assert.Equal(t, "missing_payload_hashes", diagnostics[1].Name())
		assert.Equal(t, "[0100000000000000000000000000000000000000000000000000000000000000]", diagnostics[1].String())
	})

	t.Run("error - missing payloads (doesn't panic/fail)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		payloadCollector := NewMockmissingPayloadCollector(ctrl)
		payloadCollector.EXPECT().findMissingPayloads().Return(nil, errors.New("oops"))

		instance := NewProtocol(nil, nil, nil, nil).(*protocol)
		instance.missingPayloadCollector = payloadCollector
		diagnostics := instance.Diagnostics()
		assert.Equal(t, "missing_payload_hashes", diagnostics[1].Name())
		assert.Empty(t, "", diagnostics[1].String())
	})
}
