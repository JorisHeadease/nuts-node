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

package network

import (
	"github.com/nuts-foundation/nuts-node/crypto/hash"
	"github.com/nuts-foundation/nuts-node/network/dag"
	"github.com/nuts-foundation/nuts-node/network/transport"
)

// Transactions is the interface that defines the API for creating, reading and subscribing to Nuts Network transactions.
type Transactions interface {
	// Subscribe makes a subscription for the specified transaction type. The receiver is called when a transaction
	// is received for the specified event and payload type.
	Subscribe(eventType dag.EventType, payloadType string, receiver dag.Receiver)
	// GetTransactionPayload retrieves the transaction Payload for the given transaction. If the transaction or Payload is not found
	// nil is returned.
	GetTransactionPayload(transactionRef hash.SHA256Hash) ([]byte, error)
	// GetTransaction retrieves the transaction for the given reference. If the transaction is not known, an error is returned.
	GetTransaction(transactionRef hash.SHA256Hash) (dag.Transaction, error)
	// CreateTransaction creates a new transaction according to the given spec.
	CreateTransaction(spec Template) (dag.Transaction, error)
	// ListTransactions returns all transactions known to this Network instance.
	ListTransactions() ([]dag.Transaction, error)
	// Walk walks the DAG starting at the root, calling `visitor` for every transaction.
	Walk(visitor dag.Visitor) error
	// PeerDiagnostics returns a map containing diagnostic information of the node's peers. The key contains the remote peer's ID.
	PeerDiagnostics() map[transport.PeerID]transport.Diagnostics
}
