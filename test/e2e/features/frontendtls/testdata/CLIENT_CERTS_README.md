# Client Certificates for mTLS Testing

This directory contains client certificates used for mutual TLS (mTLS) testing with the `verify-certificate-hash` annotation.

## Files

- `client.crt` / `client.key` - Valid client certificate for positive tests
- `client-invalid.crt` / `client-invalid.key` - Invalid client certificate for negative tests

## Generation Commands

### Valid Client Certificate
```bash
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout client.key -out client.crt \
  -subj "/C=US/ST=California/L=San Francisco/O=Client Inc./CN=client.example.com"
```

**SHA256 Fingerprint:**
```
E9:A5:B5:3C:8D:EA:57:2F:1A:35:D5:79:49:60:74:FD:F4:1D:7D:5E:98:BA:EB:05:CD:88:72:F6:73:BB:BF:AA
```

To calculate the fingerprint:
```bash
openssl x509 -in client.crt -noout -fingerprint -sha256 | cut -d= -f2
```

### Invalid Client Certificate
```bash
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout client-invalid.key -out client-invalid.crt \
  -subj "/C=US/ST=California/L=San Francisco/O=Invalid Client Inc./CN=invalid-client.example.com"
```

**SHA256 Fingerprint:**
```
FC:1B:6F:22:5E:D0:65:10:D2:68:A4:7E:33:63:C1:99:48:DD:19:BB:B8:A9:2C:21:16:05:F4:60:47:57:AB:B9
```

## Usage in Tests

The `client.crt` certificate hash is configured in the gateway's first mTLS listener (port 8443) `verify-certificate-hash` annotation.
The `client-invalid.crt` certificate hash is configured in the gateway's second mTLS listener (port 9443) `verify-certificate-hash` annotation.

Tests use these certificates with curl's `--cert` and `--key` flags to validate:
1. Connections succeed when the client cert hash matches the configured hash on each listener
2. Connections fail when the client cert hash doesn't match (cross-validation)
3. Connections fail when no client cert is provided to an mTLS-enabled listener
4. The regular listener (port 443) works without client certificates

