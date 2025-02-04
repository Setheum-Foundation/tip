package api

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"

	"github.com/MixinNetwork/tip/crypto"
	"github.com/MixinNetwork/tip/keeper"
	"github.com/MixinNetwork/tip/logger"
	"github.com/MixinNetwork/tip/store"
	"github.com/drand/kyber"
	"github.com/drand/kyber/pairing/bn256"
	"github.com/drand/kyber/share"
	"github.com/drand/kyber/share/dkg"
	"github.com/drand/kyber/sign/tbls"
)

type SignRequest struct {
	Data      string `json:"data"`
	Identity  string `json:"identity"`
	Signature string `json:"signature"`
}

func info(key kyber.Scalar, sigrs []dkg.Node, poly []kyber.Point) (interface{}, string) {
	signers := make([]map[string]interface{}, len(sigrs))
	for i, s := range sigrs {
		signers[i] = map[string]interface{}{
			"index":    s.Index,
			"identity": crypto.PublicKeyString(s.Public),
		}
	}
	commitments := make([]string, len(poly))
	for i, c := range poly {
		commitments[i] = crypto.PublicKeyString(c)
	}
	id := crypto.PublicKey(key)
	data := map[string]interface{}{
		"identity":    crypto.PublicKeyString(id),
		"signers":     signers,
		"commitments": commitments,
	}
	b, _ := json.Marshal(data)
	sig, _ := crypto.Sign(key, b)
	return data, hex.EncodeToString(sig)
}

func sign(key kyber.Scalar, store store.Storage, body *SignRequest, priv *share.PriShare) (interface{}, string, error) {
	res, err := keeper.Guard(store, key, body.Identity, body.Signature, body.Data)
	if err != nil {
		logger.Debug("keeper.Guard", err)
		return nil, "", ErrUnknown
	}
	if res == nil || res.Available < 1 {
		return nil, "", ErrTooManyRequest
	}
	scheme := tbls.NewThresholdSchemeOnG1(bn256.NewSuiteG2())
	partial, err := scheme.Sign(priv, res.Assignor)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, res.Nonce)
	plain := append(buf, partial...)
	plain = append(plain, res.Assignor...)
	cipher := crypto.Encrypt(res.Identity, key, plain)
	data := map[string]interface{}{
		"cipher": hex.EncodeToString(cipher),
	}
	b, _ := json.Marshal(data)
	sig, _ := crypto.Sign(key, b)
	return data, hex.EncodeToString(sig), nil
}
