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

package dag

import (
	"context"

	"github.com/nuts-foundation/nuts-node/crypto/hash"
	"github.com/nuts-foundation/nuts-node/network/storage"
	"go.etcd.io/bbolt"
)

// payloadsBucketName is the name of the Bolt bucket that holds the payloads of the transactions.
const payloadsBucketName = "payloads"

// NewBBoltPayloadStore creates a etcd/bbolt backed payload store using the given database.
func NewBBoltPayloadStore(db *bbolt.DB) PayloadStore {
	return &bboltPayloadStore{db: db}
}

type bboltPayloadStore struct {
	db *bbolt.DB
}

func (store bboltPayloadStore) IsPayloadPresent(ctx context.Context, payloadHash hash.SHA256Hash) (bool, error) {
	var result bool
	var err error
	err = store.ReadManyPayloads(ctx, func(ctx context.Context, reader PayloadReader) error {
		result, err = reader.IsPayloadPresent(ctx, payloadHash)
		return err
	})
	return result, err
}

func (store bboltPayloadStore) ReadPayload(ctx context.Context, payloadHash hash.SHA256Hash) ([]byte, error) {
	var result []byte
	var err error
	err = store.ReadManyPayloads(ctx, func(ctx context.Context, reader PayloadReader) error {
		result, err = reader.ReadPayload(ctx, payloadHash)
		return err
	})
	return result, err
}

func (store bboltPayloadStore) ReadManyPayloads(ctx context.Context, consumer func(ctx context.Context, reader PayloadReader) error) error {
	return storage.BBoltTXView(ctx, store.db, func(contextWithTX context.Context, tx *bbolt.Tx) error {
		return consumer(contextWithTX, &bboltPayloadReader{payloadsBucket: tx.Bucket([]byte(payloadsBucketName))})
	})
}

func (store bboltPayloadStore) WritePayload(ctx context.Context, payloadHash hash.SHA256Hash, data []byte) error {
	return storage.BBoltTXUpdate(ctx, store.db, func(_ context.Context, tx *bbolt.Tx) error {
		payloads, err := tx.CreateBucketIfNotExists([]byte(payloadsBucketName))
		if err != nil {
			return err
		}
		if err := payloads.Put(payloadHash.Slice(), data); err != nil {
			return err
		}
		return nil
	})
}

type bboltPayloadReader struct {
	payloadsBucket *bbolt.Bucket
}

func (reader bboltPayloadReader) IsPayloadPresent(_ context.Context, payloadHash hash.SHA256Hash) (bool, error) {
	if reader.payloadsBucket == nil {
		return false, nil
	}
	data := reader.payloadsBucket.Get(payloadHash.Slice())
	return len(data) > 0, nil
}

func (reader bboltPayloadReader) ReadPayload(_ context.Context, payloadHash hash.SHA256Hash) ([]byte, error) {
	if reader.payloadsBucket == nil {
		return nil, nil
	}
	return copyBBoltValue(reader.payloadsBucket, payloadHash.Slice()), nil
}
