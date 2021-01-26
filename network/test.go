/*
 * Copyright (C) 2021. Nuts community
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

package network

import (
	"github.com/nuts-foundation/nuts-node/crypto"
	"github.com/sirupsen/logrus"
	"path"
)

// NewTestNetworkInstance creates a new Network instance that writes it data to a test directory.
func NewTestNetworkInstance(testDirectory string) *NetworkEngine {
	config := TestNetworkConfig(testDirectory)
	newInstance := NewNetworkInstance(config, crypto.NewTestCryptoInstance(testDirectory))
	if err := newInstance.Configure(); err != nil {
		logrus.Fatal(err)
	}
	return newInstance
}

// NewTestNetworkInstance creates new network config with a test directory as data path.
func TestNetworkConfig(testDirectory string) Config {
	config := DefaultConfig()
	config.DatabaseFile = path.Join(testDirectory, "network.db")
	config.GrpcAddr = ":5555"
	config.EnableTLS = false
	config.PublicAddr = "test:5555"
	return config
}
