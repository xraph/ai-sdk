# Weaviate Installation Note

## TLS Certificate Issue

The Weaviate integration requires the following dependencies:

```bash
go get github.com/weaviate/weaviate-go-client/v4
```

However, there's currently a TLS certificate validation error on your system:

```
tls: failed to verify certificate: x509: OSStatus -26276
```

## Fix the TLS Issue

### Option 1: Disable Go Module Proxy Temporarily

```bash
export GOPROXY=direct
export GOSUMDB=off
go mod tidy
```

### Option 2: Update System Certificates (macOS)

```bash
# Update certificates
sudo security find-certificate -a -p /System/Library/Keychains/SystemRootCertificates.keychain > /tmp/certs.pem
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain /tmp/certs.pem

# Or reinstall certificates
brew install ca-certificates
```

### Option 3: Update Go

Ensure you're using Go 1.21+ which has updated certificate handling:

```bash
go version
# If older, install latest: https://go.dev/dl/
```

### Option 4: Use Corporate Proxy Settings

If behind a corporate proxy:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
export GOPRIVATE="github.com/your-org/*"
```

## Once Fixed

After fixing the TLS issue, run:

```bash
cd integrations
go mod download
go mod tidy
go test ./vectorstores/weaviate/...
```

## Alternative: Comment Out Weaviate

If you don't need Weaviate immediately, you can temporarily comment it out:

```go
// In integrations/go.mod, remove or comment:
// github.com/weaviate/weaviate-go-client/v4 v4.13.1
```

The rest of the integrations will work fine without it.

