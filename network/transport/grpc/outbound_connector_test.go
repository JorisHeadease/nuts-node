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

package grpc

import (
	"context"
	"fmt"
	"github.com/nuts-foundation/nuts-node/test"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"sync"
	"testing"
	"time"
)

func Test_connector_tryConnect(t *testing.T) {
	serverConfig := NewConfig(fmt.Sprintf("localhost:%d", test.FreeTCPPort()), "server")
	cm := NewGRPCConnectionManager(serverConfig, &TestNodeDIDResolver{}, nil)
	if !assert.NoError(t, cm.Start()) {
		return
	}
	defer cm.Stop()

	connector := createOutboundConnector(serverConfig.listenAddress, grpc.DialContext, nil, func() bool {
		return false
	}, nil)
	grpcConn, err := connector.tryConnect()
	assert.NoError(t, err)
	assert.NotNil(t, grpcConn)
	assert.Equal(t, uint32(1), connector.stats().Attempts)
}

func Test_connector_start(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		connected := make(chan struct{}, 1)
		connector := createOutboundConnector("foo", func(_ context.Context, _ string, _ ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
			return &grpc.ClientConn{}, nil
		}, nil, func() bool {
			return true
		}, func(_ *grpc.ClientConn) bool {
			connected <- struct{}{}
			return true
		})
		bo := &trackingBackoff{mux: &sync.Mutex{}}
		connector.backoff = bo
		connector.connectedBackoff = func(_ context.Context) {
			// nothing
		}

		connector.start()
		defer connector.stop()

		<-connected // wait for connected

		resetCounts, _ := bo.counts()
		assert.Equal(t, 1, resetCounts)
	})
	t.Run("not connecting when already connected", func(t *testing.T) {
		calls := make(chan struct{}, 10)
		connector := createOutboundConnector("foo", nil, nil, func() bool {
			calls <- struct{}{}
			return false
		}, nil)
		connector.connectedBackoff = func(_ context.Context) {
			// nothing
		}

		connector.start()

		// Wait for 3 calls, should be enough to assert the connection isn't made
		for i := 0; i < 3; i++ {
			<-calls
		}
	})
	t.Run("backoff when callback fails", func(t *testing.T) {
		connector := createOutboundConnector("foo", func(_ context.Context, _ string, _ ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
			return &grpc.ClientConn{}, nil
		}, nil, func() bool {
			return true
		}, func(_ *grpc.ClientConn) bool {
			return false
		})
		bo := &trackingBackoff{mux: &sync.Mutex{}}
		connector.backoff = bo
		connector.connectedBackoff = func(_ context.Context) {
			// nothing
		}

		connector.start()
		defer connector.stop()

		test.WaitFor(t, func() (bool, error) {
			_, backoffCount := bo.counts()
			return backoffCount >= 1, nil
		}, time.Second, "time-out while waiting for backoff to be invoked")
		resetCounts, _ := bo.counts()
		assert.Equal(t, 0, resetCounts)
	})
}

type trackingBackoff struct {
	resetCount   int
	backoffCount int
	mux          *sync.Mutex
}

func (t *trackingBackoff) counts() (int, int) {
	t.mux.Lock()
	defer t.mux.Unlock()
	return t.resetCount, t.backoffCount
}

func (t *trackingBackoff) Reset() {
	t.mux.Lock()
	defer t.mux.Unlock()
	t.resetCount++
}

func (t *trackingBackoff) Backoff() time.Duration {
	t.mux.Lock()
	defer t.mux.Unlock()
	t.backoffCount++
	return 10 * time.Millisecond // prevent spinwait
}
