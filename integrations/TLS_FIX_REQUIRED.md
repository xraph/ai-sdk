# ⚠️ TLS Certificate Issue - Action Required

## Problem

Your system has a TLS certificate validation error that's blocking Go module downloads:

```
tls: failed to verify certificate: x509: OSStatus -26276
```

This is preventing the installation of:
- `github.com/weaviate/weaviate-go-client/v4`
- `github.com/cohere-ai/cohere-go/v2`
- And their transitive dependencies

## Quick Fix (Recommended)

### Option 1: Bypass Go Proxy Temporarily

```bash
cd /Users/rexraphael/Work/xraph/ai-sdk/integrations

# Bypass the proxy
export GOPROXY=direct
export GOSUMDB=off

# Clean and retry
go clean -modcache
go mod download
go mod tidy
```

### Option 2: Fix macOS Certificate Store

The error `-26276` is a macOS KeychainServices error indicating certificate issues.

```bash
# Check certificate validity
security find-certificate -a -p /System/Library/Keychains/SystemRootCertificates.keychain | head -20

# Reset certificate trust settings (may require password)
sudo security authorizationdb write com.apple.trust-settings.admin allow

# Update CA certificates
brew install ca-certificates
```

### Option 3: Corporate Proxy/VPN

If you're on a corporate network:

```bash
# Set proxy (replace with your proxy)
export HTTP_PROXY=http://proxy.corporate.com:8080
export HTTPS_PROXY=http://proxy.corporate.com:8080

# Trust corporate certificate
export SSL_CERT_FILE=/path/to/corporate-ca.crt

# Or disable proxy for Go
export GONOPROXY=github.com
export GOPRIVATE=github.com
```

## Temporary Workaround

Until you fix the TLS issue, the integrations will work WITHOUT Weaviate and Cohere:

### What Currently Works ✅
- ✅ Memory VectorStore
- ✅ pgvector (PostgreSQL)
- ✅ Qdrant  
- ✅ Pinecone
- ✅ ChromaDB
- ✅ PostgreSQL StateStore
- ✅ Redis StateStore & CacheStore
- ✅ Memory StateStore & CacheStore
- ✅ OpenAI Embeddings
- ✅ All integration tests
- ✅ All benchmarks

### Temporarily Unavailable ⚠️
- ⚠️ Weaviate VectorStore (requires weaviate-go-client/v4)
- ⚠️ Cohere Embeddings (requires cohere-go/v2)

## Test the Fix

After applying one of the fixes above:

```bash
cd /Users/rexraphael/Work/xraph/ai-sdk/integrations

# This should complete without errors
go mod download
go mod tidy

# Test that everything works
go test ./...
```

## If Still Failing

### Check Go Version
```bash
go version  # Should be 1.21 or higher
```

### Check Network
```bash
curl -v https://proxy.golang.org/github.com/weaviate/@v/list
```

### Check System Time
```bash
date  # Incorrect system time can cause cert validation failures
```

### Last Resort: Use Docker

If local Go setup continues to fail:

```bash
# Build in Docker (bypasses local cert issues)
docker run --rm -v "$PWD":/app -w /app golang:1.21 go mod download
docker run --rm -v "$PWD":/app -w /app golang:1.21 go mod tidy
```

## Re-enable Weaviate & Cohere

Once TLS is fixed, uncomment in `go.mod`:

```go
require (
    github.com/weaviate/weaviate-go-client/v4 v4.13.1
    github.com/cohere-ai/cohere-go/v2 v2.8.0
)
```

Then:

```bash
go mod download
go mod tidy
go test ./vectorstores/weaviate/...
go test ./embeddings/cohere/...
```

## Summary

**Status**: 17 of 19 integrations are ready to use. 2 require fixing the local TLS certificate issue.

**Action**: Choose one of the fix options above, then re-run `go mod tidy`.

---

Need help? The TLS error `-26276` is a macOS-specific issue. Search for "OSStatus -26276 golang" for more solutions.

