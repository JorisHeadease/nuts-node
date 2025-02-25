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

package vcr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/nuts-foundation/nuts-node/vcr/holder"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jws"
	"gopkg.in/yaml.v2"

	ssi "github.com/nuts-foundation/go-did"
	"github.com/nuts-foundation/go-did/did"
	"github.com/nuts-foundation/go-did/vc"
	"github.com/nuts-foundation/go-leia/v2"
	"github.com/nuts-foundation/nuts-node/core"
	"github.com/nuts-foundation/nuts-node/crypto"
	"github.com/nuts-foundation/nuts-node/crypto/hash"
	"github.com/nuts-foundation/nuts-node/network"
	"github.com/nuts-foundation/nuts-node/vcr/assets"
	"github.com/nuts-foundation/nuts-node/vcr/concept"
	"github.com/nuts-foundation/nuts-node/vcr/credential"
	"github.com/nuts-foundation/nuts-node/vcr/issuer"
	"github.com/nuts-foundation/nuts-node/vcr/log"
	"github.com/nuts-foundation/nuts-node/vcr/signature"
	"github.com/nuts-foundation/nuts-node/vcr/trust"
	"github.com/nuts-foundation/nuts-node/vcr/types"
	"github.com/nuts-foundation/nuts-node/vcr/verifier"
	"github.com/nuts-foundation/nuts-node/vdr/doc"
	vdr "github.com/nuts-foundation/nuts-node/vdr/types"
)

var timeFunc = time.Now

// noSync is used to disable bbolt syncing on go-leia during tests
var noSync bool

// NewVCRInstance creates a new vcr instance with default config and empty concept registry
func NewVCRInstance(keyStore crypto.KeyStore, docResolver vdr.DocResolver, keyResolver vdr.KeyResolver, network network.Transactions) VCR {
	r := &vcr{
		config:          DefaultConfig(),
		docResolver:     docResolver,
		keyStore:        keyStore,
		keyResolver:     keyResolver,
		serviceResolver: doc.NewServiceResolver(docResolver),
		network:         network,
		registry:        concept.NewRegistry(),
	}

	return r
}

type vcr struct {
	registry        concept.Registry
	config          Config
	store           leia.Store
	keyStore        crypto.KeyStore
	docResolver     vdr.DocResolver
	keyResolver     vdr.KeyResolver
	serviceResolver doc.ServiceResolver
	ambassador      Ambassador
	network         network.Transactions
	trustConfig     *trust.Config
	issuer          issuer.Issuer
	verifier        verifier.Verifier
	holder          holder.Holder
	issuerStore     issuer.Store
	verifierStore   verifier.Store
}

func (c *vcr) Registry() concept.Reader {
	return c.registry
}

func (c vcr) Issuer() issuer.Issuer {
	return c.issuer
}

func (c vcr) Holder() holder.Holder {
	return c.holder
}

func (c *vcr) Configure(config core.ServerConfig) error {
	var err error

	// store config parameters for use in Start()
	c.config.strictMode = config.Strictmode
	c.config.datadir = config.Datadir

	issuerStorePath := path.Join(c.config.datadir, "vcr", "issued-credentials.db")
	c.issuerStore, err = issuer.NewLeiaIssuerStore(issuerStorePath)
	if err != nil {
		return err
	}

	verifierStorePath := path.Join(c.config.datadir, "vcr", "verifier-store.db")
	c.verifierStore, err = verifier.NewLeiaVerifierStore(verifierStorePath)
	if err != nil {
		return err
	}

	// Create the JSON-LD Context loader
	allowExternalCalls := !config.Strictmode
	contextLoader, err := signature.NewContextLoader(allowExternalCalls)

	publisher := issuer.NewNetworkPublisher(c.network, c.docResolver, c.keyStore)
	c.issuer = issuer.NewIssuer(c.issuerStore, publisher, c.docResolver, c.keyStore, contextLoader)
	c.verifier = verifier.NewVerifier(c.verifierStore, c.keyResolver, contextLoader)

	c.ambassador = NewAmbassador(c.network, c, c.verifier)

	c.holder = holder.New(c.keyResolver, c.keyStore, c.verifier, contextLoader)

	// load VC concept templates
	if err = c.loadTemplates(); err != nil {
		return err
	}

	// load trusted issuers
	tcPath := path.Join(config.Datadir, "vcr", "trusted_issuers.yaml")
	c.trustConfig = trust.NewConfig(tcPath)

	return c.trustConfig.Load()
}

func (c *vcr) credentialsDBPath() string {
	return path.Join(c.config.datadir, "vcr", "credentials.db")
}

func (c *vcr) Migrate() error {
	// the migration to go-leia V2 needs a fresh DB
	// The DAG is rewalked so all entries are added
	// just delete
	// TODO remove after all parties in development network have migrated.
	err := os.Remove(c.credentialsDBPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (c *vcr) Start() error {
	var err error

	// setup DB connection
	if c.store, err = leia.NewStore(c.credentialsDBPath(), noSync); err != nil {
		return err
	}

	// init indices
	if err = c.initIndices(); err != nil {
		return err
	}

	// start listening for new credentials
	c.ambassador.Configure()

	return nil
}

func (c *vcr) Shutdown() error {
	err := c.issuerStore.Close()
	if err != nil {
		log.Logger().Errorf("Unable to close issuer store: %v", err)
	}
	err = c.verifierStore.Close()
	if err != nil {
		log.Logger().Errorf("Unable to close verifier store: %v", err)
	}
	return c.store.Close()
}

func (c *vcr) loadTemplates() error {
	list, err := fs.Glob(assets.Assets, "**/*.config.yaml")
	if err != nil {
		return err
	}

	for _, f := range list {
		bytes, err := assets.Assets.ReadFile(f)
		if err != nil {
			return err
		}
		config := concept.Config{}
		err = yaml.Unmarshal(bytes, &config)
		if err != nil {
			return err
		}

		if err = c.registry.Add(config); err != nil {
			return err
		}
	}

	return nil
}

func whitespaceOrExactTokenizer(text string) (tokens []string) {
	tokens = leia.WhiteSpaceTokenizer(text)
	tokens = append(tokens, text)

	return
}

func (c *vcr) initIndices() error {
	for _, config := range c.registry.Concepts() {
		collection := c.store.Collection(config.CredentialType)
		for _, index := range config.Indices {
			var leiaParts []leia.FieldIndexer

			for _, iParts := range index.Parts {
				options := make([]leia.IndexOption, 0)
				if iParts.Alias != nil {
					options = append(options, leia.AliasOption(*iParts.Alias))
				}
				if iParts.Tokenizer != nil {
					tokenizer := strings.ToLower(*iParts.Tokenizer)
					switch tokenizer {
					case "whitespaceorexact":
						options = append(options, leia.TokenizerOption(whitespaceOrExactTokenizer))
					case "whitespace":
						options = append(options, leia.TokenizerOption(leia.WhiteSpaceTokenizer))
					default:
						return fmt.Errorf("unknown tokenizer %s for %s", *iParts.Tokenizer, config.CredentialType)
					}
				}
				if iParts.Transformer != nil {
					transformer := strings.ToLower(*iParts.Transformer)
					switch transformer {
					case "cologne":
						options = append(options, leia.TransformerOption(concept.CologneTransformer))
					case "lowercase":
						options = append(options, leia.TransformerOption(leia.ToLower))
					default:
						return fmt.Errorf("unknown transformer %s for %s", *iParts.Transformer, config.CredentialType)
					}
				}

				leiaParts = append(leiaParts, leia.NewFieldIndexer(iParts.JSONPath, options...))
			}

			leiaIndex := leia.NewIndex(index.Name, leiaParts...)
			log.Logger().Debugf("Adding index %s to %s using: %v", index.Name, config.CredentialType, leiaIndex)

			if err := collection.AddIndex(leiaIndex); err != nil {
				return err
			}
		}
	}

	// revocation indices
	rIndex := c.revocationIndex()
	return rIndex.AddIndex(leia.NewIndex("index_subject", leia.NewFieldIndexer(concept.SubjectField)))
}

func (c *vcr) Name() string {
	return moduleName
}

func (c *vcr) Config() interface{} {
	return &c.config
}

// Search for matching credentials based upon a query. It returns an empty list if no matches have been found.
// The optional resolveTime will Search for credentials at that point in time.
func (c *vcr) Search(ctx context.Context, query concept.Query, allowUntrusted bool, resolveTime *time.Time) ([]vc.VerifiableCredential, error) {
	//transform query to leia query, for each template a query is returned
	queries := c.convert(query)

	var VCs = make([]vc.VerifiableCredential, 0)
	for vcType, q := range queries {
		docs, err := c.store.Collection(vcType).Find(ctx, q)
		if err != nil {
			return nil, err
		}
		for _, doc := range docs {
			foundCredential := vc.VerifiableCredential{}
			err = json.Unmarshal(doc.Bytes(), &foundCredential)
			if err != nil {
				return nil, fmt.Errorf("unable to parse credential from db: %w", err)
			}

			if err = c.Validate(foundCredential, allowUntrusted, false, resolveTime); err == nil {
				VCs = append(VCs, foundCredential)
			}
		}
	}

	return VCs, nil
}

func (c *vcr) Issue(template vc.VerifiableCredential) (*vc.VerifiableCredential, error) {
	if len(template.Type) != 1 {
		return nil, errors.New("can only issue credential with 1 type")
	}
	templateType := template.Type[0]
	templateTypeString := templateType.String()
	conceptConfig := c.registry.FindByType(templateTypeString)
	if conceptConfig == nil {
		if c.config.strictMode {
			return nil, errors.New("cannot issue non-predefined credential types in strict mode")
		}
		// non-strictmode, add the credential type to the registry
		conceptConfig = &concept.Config{
			Concept:        templateTypeString,
			CredentialType: templateTypeString,
		}
		c.registry.Add(*conceptConfig)
	}

	template.Context = append(template.Context, *credential.NutsContextURI)
	verifiableCredential, err := c.issuer.Issue(template, true, c.config.OverrideIssueAllPublic || conceptConfig.Public)

	if err != nil {
		return nil, err
	}

	// find issuer
	issuerDID, err := did.ParseDID(verifiableCredential.Issuer.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse issuer: %w", err)
	}

	if !c.trustConfig.IsTrusted(templateType, issuerDID.URI()) {
		log.Logger().WithFields(map[string]interface{}{"did": issuerDID.String(), "credential.Type": templateType}).
			Debugf("Issuer not yet trusted, adding trust for DID.")
		if err := c.Trust(templateType, issuerDID.URI()); err != nil {
			return verifiableCredential, fmt.Errorf("failed to trust issuer after issuing VC (did=%s,type=%s): %w", *issuerDID, templateType, err)
		}
	} else {
		log.Logger().WithFields(map[string]interface{}{"did": issuerDID.String(), "credential.Type": templateType}).
			Debugf("Issuer already trusted.")
	}

	return verifiableCredential, nil
}

func (c *vcr) Resolve(ID ssi.URI, resolveTime *time.Time) (*vc.VerifiableCredential, error) {
	credential, err := c.find(ID)
	if err != nil {
		return nil, err
	}

	// we don't have to check the signature, it's coming from our own store.
	if err = c.Validate(credential, false, false, resolveTime); err != nil {
		switch err {
		case types.ErrRevoked:
			return &credential, types.ErrRevoked
		case types.ErrUntrusted:
			return &credential, types.ErrUntrusted
		default:
			return nil, err
		}
	}
	return &credential, nil
}

// Validate validates the provided credential.
// The function returns nil when the credential is considered valid, the validation error otherwise.
//
// It accepts a few extra flags which configure the validation process:
// * If the allowUntrusted bool is set to true, the credential is not checked for trust.
//   This means that the validity does not depend on if the issuer-type combination is set to be trusted on this node.
// * If the checkSignature is set to false, the signature will not be checked.
//   If it is set to true, the signature must compute. Also, the used signing key must be valid at the validAt time.
//   A signing key is considered valid if the issuer AND at least one (if any) of its controllers was active at the validAt time.
// * If the validAt is not provided, validAt will be set to the current time.
//
// In addition to the signing key time checks, the following checks will be performed:
// * The ID fields must be set
// * The credential is not revoked (note: the revocation state is currently time independent)
// * The type must contain exactly one type in addition to the default `VerifiableCredential` type.
// * The issuanceDate must be before the validAt.
// * The expirationDate must be after the validAt.
func (c *vcr) Validate(credential vc.VerifiableCredential, allowUntrusted bool, checkSignature bool, validAt *time.Time) error {
	if credential.ID == nil {
		return errors.New("verifying a credential requires it to have a valid ID")
	}

	if validAt == nil {
		now := timeFunc()
		validAt = &now
	}

	// check for old api
	revoked, err := c.isRevoked(*credential.ID)
	if revoked {
		return types.ErrRevoked
	}
	if err != nil {
		return err
	}

	// check for new api
	revoked, err = c.verifier.IsRevoked(*credential.ID)
	if revoked {
		return types.ErrRevoked
	}
	if err != nil {
		return err
	}

	if checkSignature {
		// check if the issuer was valid at the given time. (not deactivated, or deactivated controller)
		issuerDID, _ := did.ParseDID(credential.Issuer.String())
		_, _, err = c.docResolver.Resolve(*issuerDID, &vdr.ResolveMetadata{ResolveTime: validAt, AllowDeactivated: false})
		if err != nil {
			return fmt.Errorf("could not check validity of signing key: %w", err)
		}
	}

	if !allowUntrusted {
		trusted := c.isTrusted(credential)
		if !trusted {
			return types.ErrUntrusted
		}
	}

	// perform the rest of the verification steps
	return c.verifier.Verify(credential, allowUntrusted, checkSignature, validAt)
}

func (c *vcr) isTrusted(credential vc.VerifiableCredential) bool {
	for _, t := range credential.Type {
		if c.trustConfig.IsTrusted(t, credential.Issuer) {
			return true
		}
	}

	return false
}

// find only returns a VC from storage, it does not tell anything about validity
func (c *vcr) find(ID ssi.URI) (vc.VerifiableCredential, error) {
	credential := vc.VerifiableCredential{}
	qp := leia.Eq(concept.IDField, ID.String())
	q := leia.New(qp)

	ctx, cancel := context.WithTimeout(context.Background(), maxFindExecutionTime)
	defer cancel()
	for _, t := range c.registry.Concepts() {
		docs, err := c.store.Collection(t.CredentialType).Find(ctx, q)
		if err != nil {
			return credential, err
		}
		if len(docs) > 0 {
			// there can be only one
			err = json.Unmarshal(docs[0].Bytes(), &credential)
			if err != nil {
				return credential, fmt.Errorf("unable to parse credential from db: %w", err)
			}

			return credential, nil
		}
	}

	return credential, types.ErrNotFound
}

// Revoke checks if the credential is already revoked, if not, it instructs the issuer role to revoke the credential.
func (c *vcr) Revoke(credentialID ssi.URI) (*credential.Revocation, error) {
	isRevoked, err := c.verifier.IsRevoked(credentialID)
	if err != nil {
		return nil, fmt.Errorf("error while checking revocation status: %w", err)
	}
	if isRevoked {
		return nil, core.PreconditionFailedError("credential already revoked")
	}

	return c.issuer.Revoke(credentialID)
}

func (c *vcr) Trust(credentialType ssi.URI, issuer ssi.URI) error {
	err := c.trustConfig.AddTrust(credentialType, issuer)
	if err != nil {
		log.Logger().Infof("Added trust for Verifiable Credential issuer (type=%s, issuer=%s)", credentialType, issuer)
	}
	return err
}

func (c *vcr) Untrust(credentialType ssi.URI, issuer ssi.URI) error {
	err := c.trustConfig.RemoveTrust(credentialType, issuer)
	if err != nil {
		log.Logger().Infof("Untrusted for Verifiable Credential issuer (type=%s, issuer=%s)", credentialType, issuer)
	}
	return err
}

func (c *vcr) Trusted(credentialType ssi.URI) ([]ssi.URI, error) {
	concepts := c.registry.Concepts()

	for _, concept := range concepts {
		if concept.CredentialType == credentialType.String() {
			return c.trustConfig.List(credentialType), nil
		}
	}

	log.Logger().Warnf("No credential with type %s configured", credentialType.String())

	return nil, types.ErrInvalidCredential
}

func (c *vcr) Untrusted(credentialType ssi.URI) ([]ssi.URI, error) {
	trustMap := make(map[string]bool)
	untrusted := make([]ssi.URI, 0)
	for _, trusted := range c.trustConfig.List(credentialType) {
		trustMap[trusted.String()] = true
	}

	// match all keys
	query := leia.New(leia.Prefix(concept.IssuerField, ""))

	// use type specific collection
	collection := c.store.Collection(credentialType.String())

	// for each key: add to untrusted if not present in trusted
	err := collection.IndexIterate(query, func(key []byte, value []byte) error {
		// we iterate over all issuers->reference pairs
		issuer := string(key)
		if _, ok := trustMap[issuer]; !ok {
			u, err := ssi.ParseURI(issuer)
			if err != nil {
				return err
			}
			trustMap[issuer] = true
			untrusted = append(untrusted, *u)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, leia.ErrNoIndex) {
			log.Logger().Warnf("No index with field 'issuer' found for %s", credentialType.String())

			return nil, types.ErrInvalidCredential
		}
		return nil, err
	}

	return untrusted, nil
}

func (c *vcr) Get(conceptName string, allowUntrusted bool, subject string) (concept.Concept, error) {
	q, err := c.Registry().QueryFor(conceptName)
	if err != nil {
		return nil, err
	}

	q.AddClause(concept.Eq(concept.SubjectField, subject))

	ctx, cancel := context.WithTimeout(context.Background(), maxFindExecutionTime)
	defer cancel()
	// finding a VC that backs a concept always occurs in the present, so no resolveTime needs to be passed.
	vcs, err := c.Search(ctx, q, allowUntrusted, nil)
	if err != nil {
		return nil, err
	}

	if len(vcs) == 0 {
		return nil, types.ErrNotFound
	}

	// multiple valids, use first one
	return c.Registry().Transform(conceptName, vcs[0])
}

func (c *vcr) SearchConcept(ctx context.Context, conceptName string, allowUntrusted bool, queryParams map[string]string) ([]concept.Concept, error) {
	query, err := c.registry.QueryFor(conceptName)
	if err != nil {
		return nil, err
	}

	for key, value := range queryParams {
		query.AddClause(concept.Prefix(key, value))
	}

	results, err := c.Search(ctx, query, allowUntrusted, nil)
	if err != nil {
		return nil, err
	}

	var transformedResults = make([]concept.Concept, len(results))
	for i, result := range results {
		transformedResult, err := c.registry.Transform(conceptName, result)
		if err != nil {
			return nil, err
		}
		transformedResults[i] = transformedResult
	}
	return transformedResults, nil
}

func (c *vcr) verifyRevocation(r credential.Revocation) error {
	// it must have valid content
	if err := credential.ValidateRevocation(r); err != nil {
		return err
	}

	// issuer must be the same as vc issuer
	subject := r.Subject
	subject.Fragment = ""
	if subject != r.Issuer {
		return errors.New("issuer of revocation is not the same as issuer of credential")
	}

	// create correct challenge for verification
	payload := generateRevocationChallenge(r)

	// extract proof, can't fail, already done in generateRevocationChallenge
	splittedJws := strings.Split(r.Proof.Jws, "..")
	if len(splittedJws) != 2 {
		return errors.New("invalid 'jws' value in proof")
	}
	sig, err := base64.RawURLEncoding.DecodeString(splittedJws[1])
	if err != nil {
		return err
	}

	// check if key is of issuer
	vm := r.Proof.VerificationMethod
	vm.Fragment = ""
	if vm != r.Issuer {
		return errors.New("verification method is not of issuer")
	}

	// find key
	pk, err := c.keyResolver.ResolveSigningKey(r.Proof.VerificationMethod.String(), &r.Date)
	if err != nil {
		return err
	}

	// the proof must be correct
	verifier, _ := jws.NewVerifier(jwa.ES256)
	// the jws lib can't do this for us, so we concat hdr with payload for verification
	challenge := fmt.Sprintf("%s.%s", splittedJws[0], payload)
	if err = verifier.Verify([]byte(challenge), sig, pk); err != nil {
		return err
	}

	return nil
}

func (c *vcr) isRevoked(ID ssi.URI) (bool, error) {
	qp := leia.Eq(concept.SubjectField, ID.String())
	q := leia.New(qp)

	gIndex := c.revocationIndex()
	ctx, cancel := context.WithTimeout(context.Background(), maxFindExecutionTime)
	defer cancel()
	docs, err := gIndex.Find(ctx, q)
	if err != nil {
		return false, err
	}

	if len(docs) >= 1 {
		return true, nil
	}

	return false, nil
}

// convert returns a map of credential type to query
// credential type is then used as collection input
func (c *vcr) convert(query concept.Query) map[string]leia.Query {
	var qs = make(map[string]leia.Query, 0)

	for _, tq := range query.Parts() {
		var q leia.Query
		for _, clause := range tq.Clauses {
			var qp leia.QueryPart

			switch clause.Type() {
			case concept.EqType:
				qp = leia.Eq(clause.Key(), clause.Seek())
			case concept.PrefixType:
				qp = leia.Prefix(clause.Key(), clause.Seek())
			default:
				qp = leia.Range(clause.Key(), clause.Seek(), clause.Match())
			}

			if q == nil {
				q = leia.New(qp)
			} else {
				q = q.And(qp)
			}
		}
		qs[tq.CredentialType()] = q
	}

	return qs
}

func generateRevocationChallenge(r credential.Revocation) []byte {
	// without JWS
	proof := r.Proof.Proof

	// payload
	r.Proof = nil
	payload, _ := json.Marshal(r)

	// proof
	prJSON, _ := json.Marshal(proof)

	sums := append(hash.SHA256Sum(prJSON).Slice(), hash.SHA256Sum(payload).Slice()...)
	tbs := base64.RawURLEncoding.EncodeToString(sums)

	return []byte(tbs)
}
