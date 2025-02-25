package holder

import (
	"encoding/json"
	"errors"
	"fmt"
	ssi "github.com/nuts-foundation/go-did"
	"github.com/nuts-foundation/go-did/did"
	"github.com/nuts-foundation/go-did/vc"
	"github.com/nuts-foundation/nuts-node/core"
	"github.com/nuts-foundation/nuts-node/crypto"
	"github.com/nuts-foundation/nuts-node/vcr/signature"
	"github.com/nuts-foundation/nuts-node/vcr/signature/proof"
	"github.com/nuts-foundation/nuts-node/vcr/verifier"
	vdr "github.com/nuts-foundation/nuts-node/vdr/types"
	"github.com/piprate/json-gold/ld"
)

type vcHolder struct {
	keyResolver   vdr.KeyResolver
	keyStore      crypto.KeyStore
	verifier      verifier.Verifier
	contextLoader ld.DocumentLoader
}

// New creates a new Holder.
func New(keyResolver vdr.KeyResolver, keyStore crypto.KeyStore, verifier verifier.Verifier, contextLoader ld.DocumentLoader) Holder {
	return &vcHolder{
		keyResolver:   keyResolver,
		keyStore:      keyStore,
		verifier:      verifier,
		contextLoader: contextLoader,
	}
}

func (h vcHolder) BuildVP(credentials []vc.VerifiableCredential, proofOptions proof.ProofOptions, signerDID *did.DID, validateVC bool) (*vc.VerifiablePresentation, error) {
	var err error
	if signerDID == nil {
		signerDID, err = h.resolveSubjectDID(credentials)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve signer DID from VCs for creating VP: %w", err)
		}
	}

	kid, err := h.keyResolver.ResolveAssertionKeyID(*signerDID)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve assertion key for signing VP (did=%s): %w", *signerDID, err)
	}
	key, err := h.keyStore.Resolve(kid.String())
	if err != nil {
		return nil, fmt.Errorf("unable to resolve assertion key from key store for signing VP (did=%s): %w", *signerDID, err)
	}

	if validateVC {
		for _, cred := range credentials {
			err := h.verifier.Validate(cred, &proofOptions.Created)
			if err != nil {
				return nil, core.InvalidInputError("invalid credential (id=%s): %w", cred.ID, err)
			}
		}
	}

	unsignedVP := &vc.VerifiablePresentation{
		Context:              []ssi.URI{VerifiableCredentialLDContextV1},
		Type:                 []ssi.URI{VerifiablePresentationLDType},
		VerifiableCredential: credentials,
	}

	// Convert to map[string]interface{} for signing
	documentBytes, err := unsignedVP.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var document proof.Document
	err = json.Unmarshal(documentBytes, &document)
	if err != nil {
		return nil, err
	}

	// TODO: choose between different proof types (JWT or LD-Proof)
	signingResult, err := proof.
		NewLDProof(proofOptions).
		Sign(document, signature.JSONWebSignature2020{ContextLoader: h.contextLoader}, key)
	if err != nil {
		return nil, fmt.Errorf("unable to sign VP with LD proof: %w", err)
	}

	var signedVP vc.VerifiablePresentation
	signedVPData, _ := json.Marshal(signingResult)
	err = json.Unmarshal(signedVPData, &signedVP)
	if err != nil {
		return nil, err
	}

	return &signedVP, nil
}

func (h vcHolder) resolveSubjectDID(credentials []vc.VerifiableCredential) (*did.DID, error) {
	type credentialSubject struct {
		ID did.DID `json:"id"`
	}
	var subjectID did.DID
	for _, credential := range credentials {
		var subjects []credentialSubject
		err := credential.UnmarshalCredentialSubject(&subjects)
		if err != nil || len(subjects) != 1 {
			return nil, errors.New("not all VCs contain credentialSubject.id")
		}
		subject := subjects[0]
		if !subjectID.Empty() && !subjectID.Equals(subject.ID) {
			return nil, errors.New("not all VCs have the same credentialSubject.id")
		}
		subjectID = subject.ID
	}

	if subjectID.Empty() {
		return nil, errors.New("could not resolve subject DID from VCs")
	}

	return &subjectID, nil
}
