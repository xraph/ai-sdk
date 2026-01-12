module github.com/xraph/ai-sdk/integrations

go 1.25.0

require (
	github.com/qdrant/go-client v1.16.2
	github.com/xraph/ai-sdk v0.1.0
	github.com/xraph/go-utils v0.1.0
	google.golang.org/grpc v1.76.0
)

require (
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/oapi-codegen/runtime v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250804133106-a7a43d27e69b // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251111163417-95abcf5c77ba // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	// Vector stores
	github.com/jackc/pgx/v5 v5.5.1
	github.com/pinecone-io/go-pinecone v1.1.0

	// State & Cache stores
	github.com/redis/go-redis/v9 v9.4.0

	// Embeddings
	github.com/sashabaranov/go-openai v1.20.0

	// Testing
	github.com/stretchr/testify v1.11.1
)

replace github.com/xraph/ai-sdk => ../

replace github.com/xraph/go-utils => ../../go-utils
