package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"

	"github.com/murar8/local-jwks-server/internal/config"
	"github.com/murar8/local-jwks-server/internal/handler"
	"github.com/murar8/local-jwks-server/internal/jwkutil"
)

var h handler.Handler
var key jwk.Key

func init() {
	cfg := config.JWK{Alg: jwa.RS256, Use: jwk.ForSignature}

	var err error
	key, err = jwkutil.GenerateKey(&cfg)
	if err != nil {
		panic(err)
	}

	h = handler.New(key)
}

func TestHandleJWKS(t *testing.T) {
	t.Run(("serializes the supplied JWK set"), func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
		w := httptest.NewRecorder()

		h.HandleJWKS(w, req)

		res := w.Result()
		defer res.Body.Close()
		var data map[string][]map[string]string
		err := json.NewDecoder(res.Body).Decode(&data)

		assert.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.Equal(t, "RSA", data["keys"][0]["kty"])
		assert.Equal(t, "sig", data["keys"][0]["use"])
		assert.Equal(t, "RS256", data["keys"][0]["alg"])
	})
}

func TestHandleSign(t *testing.T) {
	t.Run(("generates a signed jwt with the provided payload"), func(t *testing.T) {
		payload := map[string]string{
			"sub":    "me",
			"custom": "value",
		}

		body, err := json.Marshal(payload)
		if err != nil {
			panic(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/jwt/sign", bytes.NewReader(body))
		w := httptest.NewRecorder()

		h.HandleSign(w, req)

		res := w.Result()
		defer res.Body.Close()
		var data map[string]string
		err = json.NewDecoder(res.Body).Decode(&data)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, res.StatusCode)

		pk, err := key.PublicKey()
		if err != nil {
			panic(err)
		}

		parsed, err := jwt.Parse([]byte(data["jwt"]), jwt.WithKey(key.Algorithm(), pk))

		assert.NoError(t, err)
		assert.Equal(t, "me", parsed.Subject())
		assert.Equal(t, "value", parsed.PrivateClaims()["custom"])
	})
}
