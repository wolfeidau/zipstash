package ciauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/require"
	pingv1 "github.com/wolfeidau/zipstash/api/gen/proto/go/ping/v1"
	"github.com/wolfeidau/zipstash/api/gen/proto/go/ping/v1/pingv1connect"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

const (
	audience = "zipstash.test.com"
	issuer   = "http://test.com"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	assert := require.New(t)

	jwkey := GenerateRsaJwk(t)

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/.well-known/keys", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request: %v", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(GetRawPublicKey(t, jwkey))
	})

	endpoints := map[string]OIDCProvider{
		"test": {
			Issuer:  srv.URL,
			JWKSURL: fmt.Sprintf("%s/.well-known/keys", srv.URL),
		},
	}

	ov, err := NewOIDCValidator(context.TODO(), endpoints)
	assert.NoError(err)

	issueDate := time.Now()

	tok, err := jwt.NewBuilder().
		Issuer(srv.URL).
		Audience([]string{audience}).
		Subject("organization:abc123:pipeline:zipstash:ref:refs/heads/feat_buildkite_pipeline:commit:abc123456:step:test").
		IssuedAt(issueDate).
		NotBefore(issueDate).
		Expiration(issueDate.Add(time.Hour)).
		Build()
	assert.NoError(err)

	rawToken, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), jwkey))
	assert.NoError(err)

	fmt.Println(endpoints["test"])

	validToken, err := ov.ValidateToken(context.TODO(), "test", string(rawToken), audience)
	assert.NoError(err)
	issuer, _ := validToken.Issuer()
	assert.Equal(issuer, srv.URL)
}

func GenerateRsaKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func GenerateRsaJwk(t *testing.T) jwk.Key {
	key, err := GenerateRsaKey()
	if err != nil {
		t.Fatal(err)
	}

	k, err := jwk.Import(key)
	if err != nil {
		t.Fatal(err)
	}

	_ = k.Set(jwk.KeyIDKey, "mykey")
	_ = k.Set(jwk.AlgorithmKey, jwa.RS256())

	return k
}

func GetRawPublicKey(t *testing.T, jwkey jwk.Key) []byte {
	pub, err := jwkey.PublicKey()
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(pub)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func TestEmptyUnaryInterceptorFunc(t *testing.T) {
	// t.Parallel()

	assert := require.New(t)

	jwkey := GenerateRsaJwk(t)

	mux := http.NewServeMux()
	oidcserver := httptest.NewServer(mux)
	defer oidcserver.Close()

	mux.HandleFunc("/.well-known/keys", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request: %v", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(GetRawPublicKey(t, jwkey))
	})

	endpoints := map[string]OIDCProvider{
		"buildkite": {
			Issuer:  oidcserver.URL,
			JWKSURL: fmt.Sprintf("%s/.well-known/keys", oidcserver.URL),
		},
	}

	ov, err := NewOIDCValidator(context.TODO(), endpoints)
	assert.NoError(err)

	issueDate := time.Now()

	tok, err := jwt.NewBuilder().
		Issuer(oidcserver.URL).
		Audience([]string{audience}).
		Subject("organization:abc123:pipeline:zipstash:ref:refs/heads/feat_buildkite_pipeline:commit:abc123456:step:test").
		IssuedAt(issueDate).
		NotBefore(issueDate).
		Expiration(issueDate.Add(time.Hour)).
		Build()
	assert.NoError(err)

	rawToken, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), jwkey))
	assert.NoError(err)

	_, err = trace.NewProvider(context.Background(), "buildkite", "0.0.1")
	assert.NoError(err)

	interceptor := NewOIDCAuthInterceptor(audience, ov)
	mux.Handle(pingv1connect.NewPingServiceHandler(&pingServer{
		PingFunc: func(ctx context.Context, req *connect.Request[pingv1.PingRequest]) (*connect.Response[pingv1.PingResponse], error) {

			identity := GetOIDCIdentity(ctx)
			assert.NotNil(identity)
			assert.Equal(identity.Provider(), "buildkite")
			assert.Equal(identity.Subject(), "organization:abc123:pipeline:zipstash:ref:refs/heads/feat_buildkite_pipeline:commit:abc123456:step:test")

			return connect.NewResponse(&pingv1.PingResponse{
				Text:   req.Msg.Text,
				Number: req.Msg.Number,
			}), nil
		},
	}, connect.WithInterceptors(interceptor)))
	connectserver := httptest.NewServer(mux)
	connectClient := pingv1connect.NewPingServiceClient(connectserver.Client(), connectserver.URL, connect.WithInterceptors(interceptor))

	req := connect.NewRequest(&pingv1.PingRequest{
		Text: "hello",
	})

	req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", rawToken))
	req.Header().Set("X-Provider", "buildkite")

	_, err = connectClient.Ping(context.Background(), req)
	fmt.Printf("err: %v", err)
	assert.Nil(err)
}

type pingServer struct {
	PingFunc func(context.Context, *connect.Request[pingv1.PingRequest]) (*connect.Response[pingv1.PingResponse], error)
}

func (ps *pingServer) Ping(
	ctx context.Context,
	req *connect.Request[pingv1.PingRequest],
) (*connect.Response[pingv1.PingResponse], error) {
	return ps.PingFunc(ctx, req)
}
