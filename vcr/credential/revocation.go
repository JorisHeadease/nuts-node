/*
 * Nuts node
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

package credential

import (
	"encoding/json"
	"time"

	ssi "github.com/nuts-foundation/go-did"
	"github.com/nuts-foundation/go-did/vc"
)

// Revocation defines a proof that a VC has been revoked by it's issuer.
type Revocation struct {
	//Context []ssi.URI `json:"@context"`
	// Issuer refers to the party that issued the credential
	Issuer ssi.URI `json:"issuer"`
	// Subject refers to the VC that is revoked
	Subject ssi.URI `json:"subject"`
	// Reason describes why the VC has been revoked
	Reason string `json:"reason,omitempty"`
	// Date is a rfc3339 formatted datetime.
	Date  time.Time                      `json:"date"`
	Proof []vc.JSONWebSignature2020Proof `json:"proof,omitempty"`
}

// RevocationWithLegacyProof contains the original struct which must stay the same, otherwise the json marshalling breaks the order of fields and thus the signature.
type RevocationWithLegacyProof struct {
	// Issuer refers to the party that issued the credential
	Issuer ssi.URI `json:"issuer"`
	// Subject refers to the VC that is revoked
	Subject ssi.URI `json:"subject"`
	// Reason describes why the VC has been revoked
	Reason string `json:"reason,omitempty"`
	// Date is a rfc3339 formatted datetime.
	Date time.Time `json:"date"`
	// Proof contains the single proof value
	Proof *vc.JSONWebSignature2020Proof `json:"proof,omitempty"`
}

// BuildRevocation generates a revocation based on the credential
func BuildRevocation(credential vc.VerifiableCredential) Revocation {
	return Revocation{
		Issuer:  credential.Issuer,
		Subject: *credential.ID,
		Date:    nowFunc(),
	}
}

// MarshalJSON marshalls the revocation into valid json.
// It uses an array for proof when there are multiple proofs, but a single object value when there is only one proof.
func (r Revocation) MarshalJSON() ([]byte, error) {
	type alias Revocation
	tmp := alias(r)
	proof := r.Proof
	r.Proof = nil
	revocationAsMap := map[string]interface{}{}
	rBytes, _ := json.Marshal(tmp)
	if err := json.Unmarshal(rBytes, &revocationAsMap); err != nil {
		return []byte{}, err
	}
	if len(proof) > 0 {
		if len(proof) == 1 {
			revocationAsMap["proof"] = proof[0]
		} else {
			revocationAsMap["proof"] = proof
		}
	}
	return json.Marshal(revocationAsMap)
}

// UnmarshalJSON unmarshalls the revocation from valid json.
// It accepts both an object value as an array value for proof.
func (r *Revocation) UnmarshalJSON(b []byte) error {
	asMap := map[string]interface{}{}
	err := json.Unmarshal(b, &asMap)
	if err != nil {
		return err
	}

	proof, hasProof := asMap["proof"]
	if hasProof {
		if _, isSlice := proof.([]interface{}); isSlice {
			// ok
		} else if val, isMap := proof.(map[string]interface{}); isMap {
			// convert to slice
			asMap["proof"] = []interface{}{val}
		}
	}

	mapBytes, err := json.Marshal(asMap)
	if err != nil {
		return err
	}

	type alias Revocation
	tmp := alias{}
	if err := json.Unmarshal(mapBytes, &tmp); err != nil {
		return err
	}

	*r = (Revocation)(tmp)
	return nil
}

//func (r *RevocationWithLDProof) SetProof(proof vc.JSONWebSignature2020Proof) {
//	r.Proof = []vc.JSONWebSignature2020Proof{proof}
//}

// ValidateRevocation checks if a revocation record contains the required fields and if fields have the correct value.
func ValidateRevocation(r Revocation) error {
	if r.Subject.String() == "" || r.Subject.Fragment == "" {
		return failure("'subject' is required and requires a valid fragment")
	}

	if r.Issuer.String() == "" {
		return failure("'issuer' is required")
	}

	if r.Date.IsZero() {
		return failure("'date' is required")
	}

	if len(r.Proof) == 0 {
		return failure("'proof' is required")
	}

	if len(r.Proof) > 1 {
		return failure("'proof' can only contain one value")
	}

	return nil
}
