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

package doc

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/nuts-foundation/go-did/did"
	"github.com/nuts-foundation/nuts-node/vdr/types"
	"github.com/stretchr/testify/assert"

	nutsCrypto "github.com/nuts-foundation/nuts-node/crypto"
)

var jwkString = `{"crv":"P-256","kid":"did:nuts:3gU9z3j7j4VCboc3qq3Vc5mVVGDNGjfg32xokeX8c8Zn#J9O6wvqtYOVwjc8JtZ4aodRdbPv_IKAjLkEq9uHlDdE","kty":"EC","x":"Qn6xbZtOYFoLO2qMEAczcau9uGGWwa1bT+7JmAVLtg4=","y":"d20dD0qlT+d1djVpAfrfsAfKOUxKwKkn1zqFSIuJ398="},"type":"JsonWebKey2020"}`

func TestDefaultCreationOptions(t *testing.T) {
	ops := DefaultCreationOptions()

	assert.True(t, ops.AssertionMethod)
	assert.False(t, ops.Authentication)
	assert.False(t, ops.CapabilityDelegation)
	assert.True(t, ops.CapabilityInvocation)
	assert.True(t, ops.KeyAgreement)
	assert.True(t, ops.SelfControl)
	assert.Empty(t, ops.Controllers)
}

func TestCreator_Create(t *testing.T) {
	defaultOptions := DefaultCreationOptions()

	t.Run("ok", func(t *testing.T) {
		kc := newMockKeyCreator()
		creator := Creator{KeyStore: kc}
		t.Run("defaults", func(t *testing.T) {
			doc, key, err := creator.Create(defaultOptions)
			assert.NoError(t, err, "create should not return an error")
			assert.NotNil(t, doc, "create should return a document")
			assert.NotNil(t, key, "create should return a Key")
			assert.Equal(t, "did:nuts:3gU9z3j7j4VCboc3qq3Vc5mVVGDNGjfg32xokeX8c8Zn", doc.ID.String(), "the DID Doc should have the expected id")
			assert.Len(t, doc.VerificationMethod, 1, "it should have one verificationMethod")
			assert.Equal(t, "did:nuts:3gU9z3j7j4VCboc3qq3Vc5mVVGDNGjfg32xokeX8c8Zn#J9O6wvqtYOVwjc8JtZ4aodRdbPv_IKAjLkEq9uHlDdE", doc.VerificationMethod[0].ID.String(),
				"verificationMethod should have the correct id")
			assert.Len(t, doc.CapabilityInvocation, 1, "it should have 1 CapabilityInvocation")
			assert.Equal(t, doc.CapabilityInvocation[0].VerificationMethod, doc.VerificationMethod[0], "the assertionMethod should be a pointer to the verificationMethod")
			assert.Len(t, doc.AssertionMethod, 1, "it should have 1 AssertionMethod")
		})

		t.Run("all keys", func(t *testing.T) {
			ops := types.DIDCreationOptions{
				AssertionMethod:      true,
				Authentication:       true,
				CapabilityDelegation: true,
				CapabilityInvocation: true,
				KeyAgreement:         true,
				SelfControl:          true,
			}
			doc, _, err := creator.Create(ops)

			if !assert.NoError(t, err) {
				return
			}

			assert.Len(t, doc.AssertionMethod, 1)
			assert.Len(t, doc.Authentication, 1)
			assert.Len(t, doc.CapabilityDelegation, 1)
			assert.Len(t, doc.CapabilityInvocation, 1)
			assert.Len(t, doc.KeyAgreement, 1)
		})

		t.Run("extra controller", func(t *testing.T) {
			c, _ := did.ParseDID("did:nuts:controller")
			ops := types.DIDCreationOptions{
				AssertionMethod:      true,
				CapabilityInvocation: true,
				SelfControl:          true,
				Controllers:          []did.DID{*c},
			}
			doc, _, err := creator.Create(ops)

			if !assert.NoError(t, err) {
				return
			}

			assert.Len(t, doc.Controller, 2)
		})

		t.Run("using ephemeral key creates different keys for assertion and DID", func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			keyCreator := nutsCrypto.NewMockKeyCreator(ctrl)
			creator := Creator{KeyStore: keyCreator}

			keyCreator.EXPECT().New(gomock.Any()).DoAndReturn(func(fn nutsCrypto.KIDNamingFunc) (nutsCrypto.Key, error) {
				key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				keyName, _ := fn(key.Public())
				return nutsCrypto.TestKey{
					PrivateKey: key,
					Kid:        keyName,
				}, nil
			})

			ops := types.DIDCreationOptions{
				AssertionMethod:      true,
				Authentication:       false,
				CapabilityDelegation: false,
				CapabilityInvocation: false,
				KeyAgreement:         false,
				SelfControl:          false,
			}
			doc, docCreationKey, err := creator.Create(ops)

			if !assert.NoError(t, err) {
				return
			}

			assert.Len(t, doc.CapabilityInvocation, 0)
			assert.Len(t, doc.VerificationMethod, 1)
			assert.Len(t, doc.AssertionMethod, 1)

			subKeyID := doc.VerificationMethod[0].ID
			subKeyIDWithoutFragment := subKeyID
			subKeyIDWithoutFragment.Fragment = ""

			// Document has been created with a ephemeral key (one that won't be stored) but contains a verificationMethod
			// for different purposes (in this case assertionMethod), which is stored. The latter (`subKeyID`) must have the same DID
			// idString but a different fragment.
			assert.NotEqual(t, docCreationKey.KID(), subKeyID)
			assert.Equal(t, doc.ID.String(), subKeyIDWithoutFragment.String())
		})
	})

	t.Run("error - invalid combination", func(t *testing.T) {
		ops := types.DIDCreationOptions{
			CapabilityInvocation: false,
			SelfControl:          true,
		}
		kc := newMockKeyCreator()
		creator := Creator{KeyStore: kc}
		_, _, err := creator.Create(ops)

		assert.Equal(t, ErrInvalidOptions, err)
	})

	t.Run("error - failed to create key for selfcontrol", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockKeyStore := nutsCrypto.NewMockKeyStore(ctrl)
		defer ctrl.Finish()
		creator := Creator{KeyStore: mockKeyStore}
		mockKeyStore.EXPECT().New(gomock.Any()).Return(nil, errors.New("b00m!"))

		_, _, err := creator.Create(DefaultCreationOptions())

		if !assert.Error(t, err) {
			return
		}
		assert.EqualError(t, err, "b00m!")
	})

	t.Run("error - failed to create key for other verification method", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockKeyStore := nutsCrypto.NewMockKeyStore(ctrl)
		defer ctrl.Finish()
		creator := Creator{KeyStore: mockKeyStore}
		ops := types.DIDCreationOptions{
			AssertionMethod:      true,
			Authentication:       false,
			CapabilityDelegation: false,
			CapabilityInvocation: false,
			KeyAgreement:         false,
			SelfControl:          false,
		}
		mockKeyStore.EXPECT().New(gomock.Any()).Return(nil, errors.New("b00m!"))

		_, _, err := creator.Create(ops)

		if !assert.Error(t, err) {
			return
		}
		assert.EqualError(t, err, "b00m!")
	})
}

func Test_didKIDNamingFunc(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if assert.NoError(t, err) {
			return
		}

		keyID, err := didKIDNamingFunc(privateKey.PublicKey)
		if !assert.NoError(t, err) {
			return
		}
		assert.NotEmpty(t, keyID)
		assert.Contains(t, keyID, "did:nuts")
	})

	t.Run("ok - predefined key", func(t *testing.T) {
		pub, err := jwkToPublicKey(t, jwkString)
		if assert.NoError(t, err) {
			return
		}

		keyID, err := didKIDNamingFunc(pub)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, keyID, kid, keyID)
	})

	t.Run("nok - wrong key type", func(t *testing.T) {
		keyID, err := didKIDNamingFunc(unknownPublicKey{})
		if !assert.Error(t, err) {
			return
		}
		assert.Equal(t, "could not generate kid: invalid key type 'doc.unknownPublicKey' for jwk.New", err.Error())
		assert.Empty(t, keyID)

	})
}

func Test_didSubKIDNamingFunc(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		owningDID, _ := did.ParseDID("did:nuts:bladiebla")

		keyID, err := didSubKIDNamingFunc(*owningDID)(privateKey.PublicKey)
		if !assert.NoError(t, err) {
			return
		}
		parsedKeyID, err := did.ParseDIDURL(keyID)
		if !assert.NoError(t, err) {
			return
		}
		// Make sure the idString part of the key ID is taken from the owning DID document
		assert.Equal(t, parsedKeyID.ID, owningDID.ID)
		assert.NotEmpty(t, parsedKeyID.Fragment)
	})
}

type unknownPublicKey struct{}

func jwkToPublicKey(t *testing.T, jwkStr string) (crypto.PublicKey, error) {
	t.Helper()
	keySet, err := jwk.ParseString(jwkStr)
	if !assert.NoError(t, err) {
		return nil, err
	}
	key, _ := keySet.Get(0)
	var rawKey crypto.PublicKey
	if err = key.Raw(&rawKey); err != nil {
		return nil, err
	}
	return &rawKey, nil
}
