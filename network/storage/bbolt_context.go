/*
 * Nuts node
 * Copyright (C) 2022 Nuts community
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

package storage

import (
	"context"
	"errors"

	"go.etcd.io/bbolt"
)

var bboltTXContextKey = struct{}{}

type bboltTXCallback func(contextWithTX context.Context, tx *bbolt.Tx) error

// BBoltTXView executes the given callback in a read-only BBolt transaction. It attempts to re-use the active transaction from the given context, if there is one.
// If there's no active transaction a new one will be started.
func BBoltTXView(ctx context.Context, db *bbolt.DB, cb bboltTXCallback) error {
	err := callBBoltCallbackWithTX(ctx, db, cb, false)
	return err
}

// BBoltTXUpdate executes the given callback in a writable BBolt transaction. It attempts to re-use the active transaction from the given context, if there is one.
// If there's no active transaction a new one will be started.
// If there's an active transaction which is readonly an error will be returned.
func BBoltTXUpdate(ctx context.Context, db *bbolt.DB, cb bboltTXCallback) error {
	err := callBBoltCallbackWithTX(ctx, db, cb, true)
	return err
}

// BBoltTX returns the active TX from the context and true. If no TX is active it returns nil, false.
func BBoltTX(ctx context.Context) (*bbolt.Tx, bool) {
	tx, txIsActive := ctx.Value(bboltTXContextKey).(*bbolt.Tx)
	return tx, txIsActive
}

func callBBoltCallbackWithTX(ctx context.Context, db *bbolt.DB, cb bboltTXCallback, writable bool) error {
	tx, txIsActive := BBoltTX(ctx)
	if !txIsActive {
		// No active TX, we can simply start one here as long as we put it in the context we pass down, allowing nested BBolt database callers to re-use the transaction.
		if writable {
			return db.Update(func(tx *bbolt.Tx) error {
				return cb(context.WithValue(ctx, bboltTXContextKey, tx), tx)
			})
		}
		return db.View(func(tx *bbolt.Tx) error {
			return cb(context.WithValue(ctx, bboltTXContextKey, tx), tx)
		})
	}
	// There is an active TX we can use. We just have to check whether it's writable if we need it writable.
	if writable && !tx.Writable() {
		return errors.New("the active BBolt transaction is not writable")
	}
	return cb(ctx, tx)
}
