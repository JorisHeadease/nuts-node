/*
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

package issuer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	ssi "github.com/nuts-foundation/go-did"
	"github.com/nuts-foundation/go-did/did"
	"github.com/nuts-foundation/go-did/vc"
	"github.com/nuts-foundation/go-leia/v2"
	"github.com/nuts-foundation/nuts-node/vcr/concept"
)

// leiaIssuerStore implements the issuer Store interface. It is a simple and fast JSON store.
// Note: It can not be used in a clustered setup.
type leiaIssuerStore struct {
	issuedCredentials leia.Collection
	store             leia.Store
}

// NewLeiaIssuerStore creates a new instance of leiaIssuerStore which implements the Store interface.
func NewLeiaIssuerStore(dbPath string) (Store, error) {
	store, err := leia.NewStore(dbPath, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create leiaIssuerStore: %w", err)
	}
	collection := store.Collection("issuedCredentials")
	newLeiaStore := &leiaIssuerStore{
		issuedCredentials: collection,
		store:             store,
	}
	if err := newLeiaStore.createIndices(); err != nil {
		return nil, err
	}
	return newLeiaStore, nil
}

func (s leiaIssuerStore) StoreCredential(vc vc.VerifiableCredential) error {
	vcAsBytes, _ := json.Marshal(vc)
	doc := leia.DocumentFromBytes(vcAsBytes)
	return s.issuedCredentials.Add([]leia.Document{doc})
}

func (s leiaIssuerStore) SearchCredential(jsonLDContext ssi.URI, credentialType ssi.URI, issuer did.DID, subject *ssi.URI) ([]vc.VerifiableCredential, error) {
	query := leia.New(leia.Eq("issuer", issuer.String())).
		And(leia.Eq("type", credentialType.String())).
		And(leia.Eq("@context", jsonLDContext.String()))

	if subject != nil {

		if subjectString := subject.String(); subjectString != "" {
			query = query.And(leia.Eq("credentialSubject.id", subjectString))
		}
	}

	docs, err := s.issuedCredentials.Find(context.Background(), query)
	if err != nil {
		return nil, err
	}

	result := make([]vc.VerifiableCredential, len(docs))
	for i, doc := range docs {
		if err := json.Unmarshal(doc.Bytes(), &result[i]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s leiaIssuerStore) GetCredential(id ssi.URI) (*vc.VerifiableCredential, error) {
	query := leia.New(leia.Eq(concept.IDField, id.String()))

	results, err := s.issuedCredentials.Find(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("could not get credential by id: %w", err)
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	if len(results) > 1 {
		return nil, errors.New("found more than one credential by id")
	}
	result := results[0]
	credential := &vc.VerifiableCredential{}
	if err := json.Unmarshal(result.Bytes(), credential); err != nil {
		return credential, err
	}
	return credential, nil
}

func (s leiaIssuerStore) Close() error {
	return s.store.Close()
}

// createIndices creates the needed indices for the issued VC store
// It allows faster searching on context, type issuer and subject values.
func (s leiaIssuerStore) createIndices() error {
	searchIndex := leia.NewIndex("issuedVCs",
		leia.NewFieldIndexer("issuer"),
		leia.NewFieldIndexer("type"),
		leia.NewFieldIndexer("credentialSubject.id"),
	)

	// Index used for getting issued VCs by id
	idIndex := leia.NewIndex("issuedVCByID",
		leia.NewFieldIndexer("id"))
	return s.issuedCredentials.AddIndex(searchIndex, idIndex)
}
