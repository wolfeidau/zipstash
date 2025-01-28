package commands

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/lambda-go-extras/lambdaextras"
	lmw "github.com/wolfeidau/lambda-go-extras/middleware"
	"github.com/wolfeidau/lambda-go-extras/middleware/raw"
	zlog "github.com/wolfeidau/lambda-go-extras/middleware/zerolog"

	"github.com/wolfeidau/zipstash/api/gen/proto/go/provision/v1/provisionv1connect"
	"github.com/wolfeidau/zipstash/internal/index"
	"github.com/wolfeidau/zipstash/internal/server"
	"github.com/wolfeidau/zipstash/pkg/trace"
)

type AdminLambdaServerCmd struct {
	CacheIndexTable string `help:"table to store cache index" env:"CACHE_INDEX_TABLE"`
	TrustRemote     bool   `help:"trust remote spans"`
}

func (s *AdminLambdaServerCmd) Run(ctx context.Context, globals *Globals) error {
	tp, err := trace.NewLambdaProvider(ctx, "github.com/wolfeidau/zipstash", globals.Version)
	if err != nil {
		log.Fatal().Msgf("failed to create trace provider: %v", err)
	}
	defer func() {
		_ = tp.Shutdown(ctx)
	}()

	opts := []connect.HandlerOption{}

	awscfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ddbClientFunc := func() *dynamodb.Client {
		return dynamodb.NewFromConfig(awscfg)
	}

	store := index.MustNewStore(ctx, index.StoreConfig{
		CacheIndexTable:   s.CacheIndexTable,
		GetDynamoDBClient: ddbClientFunc,
	})

	psh := server.NewProvisionServiceHandler(store)

	mux := http.NewServeMux()
	path, handler := provisionv1connect.NewProvisionServiceHandler(psh, opts...)
	mux.Handle(path, handler)
	log.Info().Str("path", path).Msg("serving")

	flds := lmw.FieldMap{"version": "dev"}

	ch := lmw.New(
		raw.New(raw.Fields(flds)),
		zlog.New(zlog.Fields(flds)),
	).Then(lambdaextras.GenericHandler(httpadapter.NewV2(mux).ProxyWithContext))

	lambda.Start(ch)

	return nil
}
