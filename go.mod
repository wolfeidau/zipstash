module github.com/wolfeidau/zipstash

go 1.24.0

require (
	connectrpc.com/connect v1.18.1
	connectrpc.com/otelconnect v0.7.1
	github.com/alecthomas/kong v1.8.1
	github.com/aws/aws-lambda-go v1.47.0
	github.com/aws/aws-sdk-go-v2 v1.36.1
	github.com/aws/aws-sdk-go-v2/config v1.29.6
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.40.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.77.0
	github.com/aws/smithy-go v1.22.2
	github.com/awslabs/aws-lambda-go-api-proxy v0.16.2
	github.com/cenkalti/backoff/v5 v5.0.2
	github.com/google/uuid v1.6.0
	github.com/klauspost/compress v1.17.11
	github.com/lestrrat-go/httprc/v3 v3.0.0-beta1
	github.com/lestrrat-go/jwx/v3 v3.0.0-alpha1
	github.com/rs/zerolog v1.33.0
	github.com/stretchr/testify v1.10.0
	github.com/wolfeidau/dynastorev2 v0.6.0
	github.com/wolfeidau/lambda-go-extras v1.5.1
	github.com/wolfeidau/lambda-go-extras/middleware/raw v1.5.1
	github.com/wolfeidau/lambda-go-extras/middleware/zerolog v1.5.1
	github.com/wolfeidau/quickzip v1.0.1
	github.com/wolfeidau/zipstash/api v0.0.0-20250215081336-a05e8f877004
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.59.0
	go.opentelemetry.io/otel v1.34.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.34.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.34.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.34.0
	go.opentelemetry.io/otel/sdk v1.34.0
	go.opentelemetry.io/otel/trace v1.34.0
	golang.org/x/exp v0.0.0-20250210185358-939b2ce775ac
	golang.org/x/net v0.35.0
)

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.5-20250130201111-63bb56e20495.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.9 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.59 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.18.4 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.7.70 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.28 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.32 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.24.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.14 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.1 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/saracen/zipextra v0.0.0-20250129175152-f1aa42d25216 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/sync v0.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250212204824-5a70512c5d8b // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250212204824-5a70512c5d8b // indirect
	google.golang.org/grpc v1.70.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/wolfeidau/zipstash/api => ./api
