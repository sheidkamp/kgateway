# Client Certificates for mTLS Testing

This directory contains client certificates used for mutual TLS (mTLS) testing with the `verify-certificate-hash` annotation.


## Certificate Generation

The certificates were generated using the following commands:

### Client Certificate for Port 8443
```bash
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout client-8443.key -out client-8443.crt \
  -subj "/C=US/ST=California/L=San Francisco/O=Client Inc./CN=client.example.com"
```

### Client Certificate for Port 9443
```bash
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout client-9443.key -out client-9443.crt \
  -subj "/C=US/ST=California/L=San Francisco/O=Client Inc./CN=client-alt.example.com"
```

### SHA256 Fingerprints
To calculate the SHA256 fingerprint (used in `verify-certificate-hash` annotation):
```bash
# For port 8443 certificate
openssl x509 -in client-8443.crt -noout -fingerprint -sha256 | cut -d= -f2

# For port 9443 certificate
openssl x509 -in client-9443.crt -noout -fingerprint -sha256 | cut -d= -f2
```

**Calculated hashes:**
- **Port 8443 cert**: `E9:A5:B5:3C:8D:EA:57:2F:1A:35:D5:79:49:60:74:FD:F4:1D:7D:5E:98:BA:EB:05:CD:88:72:F6:73:BB:BF:AA`
- **Port 9443 cert**: `FC:1B:6F:22:5E:D0:65:10:D2:68:A4:7E:33:63:C1:99:48:DD:19:BB:B8:A9:2C:21:16:05:F4:60:47:57:AB:B9`

## Usage in Tests

### Test File Mapping
In the tests, these certificates are mounted into the curl pod from the Kubernetes secret at:
- `/etc/client-certs/client-8443.crt` / `/etc/client-certs/client-8443.key` - Certificate valid for port 8443 listener
- `/etc/client-certs/client-9443.crt` / `/etc/client-certs/client-9443.key` - Certificate valid for port 9443 listener

### Gateway Configuration
- **Port 8443** (mtls.example.com): `verify-certificate-hash` = SHA256 of the port 8443 certificate
- **Port 9443** (mtls-alt.example.com): `verify-certificate-hash` = SHA256 of the port 9443 certificate

### Test Validation
The test suite uses these certificates with curl's `--cert` and `--key` flags to validate:
1. Connections succeed when the client cert hash matches the configured hash on each listener
2. Connections fail when the client cert hash doesn't match (cross-validation between ports)
3. Connections fail when no client cert is provided to an mTLS-enabled listener
4. The regular listener (port 443) works without client certificates

