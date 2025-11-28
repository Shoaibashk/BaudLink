# Security Guide

This document describes BaudLink's security features and best practices for secure deployment.

## Transport Security

### TLS Encryption

BaudLink supports TLS encryption for secure gRPC communication:

```yaml
tls:
  enabled: true
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
```

#### Generating Certificates

For development/testing:

```bash
# Generate self-signed certificate
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/CN=localhost"
```

For production, use certificates from a trusted CA or your organization's PKI.

### Client Configuration

Clients connecting with TLS must trust the server certificate:

**Python:**

```python
import grpc

# With server certificate
with open('cert.pem', 'rb') as f:
    creds = grpc.ssl_channel_credentials(f.read())

channel = grpc.secure_channel('localhost:50051', creds)
```

**Go:**

```go
creds, _ := credentials.NewClientTLSFromFile("cert.pem", "")
conn, _ := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(creds))
```

## Port Security

### Exclusive Access

BaudLink implements exclusive port locking to prevent conflicts:

- Only one client can open a specific port at a time
- Port access is released when closed or client disconnects
- Prevents data corruption from concurrent access

### Access Control

Control which clients can access which ports through:

1. **Network Segmentation** - Limit network access to the BaudLink service
2. **Firewall Rules** - Restrict connections to trusted IPs
3. **TLS Client Certificates** - Mutual TLS for client authentication

## Network Security

### Binding Address

For local-only access:

```yaml
server:
  grpc_address: "127.0.0.1:50051"
```

For network access (with appropriate firewall rules):

```yaml
server:
  grpc_address: "0.0.0.0:50051"
```

### Firewall Configuration

**Linux (iptables):**

```bash
# Allow from specific subnet
iptables -A INPUT -p tcp --dport 50051 -s 192.168.1.0/24 -j ACCEPT
iptables -A INPUT -p tcp --dport 50051 -j DROP
```

**Windows Firewall:**

```powershell
# Allow from specific subnet
New-NetFirewallRule -DisplayName "BaudLink" -Direction Inbound `
  -LocalPort 50051 -Protocol TCP -Action Allow -RemoteAddress 192.168.1.0/24
```

## Deployment Best Practices

### 1. Use TLS in Production

Always enable TLS when the service is accessible over a network:

```yaml
tls:
  enabled: true
  cert_file: "/etc/baudlink/certs/server.crt"
  key_file: "/etc/baudlink/certs/server.key"
```

### 2. Minimal Network Exposure

- Bind to localhost when possible
- Use network segmentation for IoT/device networks
- Place behind a reverse proxy for additional controls

### 3. File Permissions

Secure configuration and certificate files:

```bash
# Linux
chmod 600 /etc/baudlink/agent.yaml
chmod 600 /etc/baudlink/certs/*
chown root:root /etc/baudlink/*
```

```powershell
# Windows
icacls "C:\ProgramData\BaudLink\agent.yaml" /inheritance:r /grant:r "SYSTEM:F" "Administrators:F"
```

### 4. Service Account

Run BaudLink as a dedicated service account with minimal privileges:

**Linux:**

```bash
# Create service user
useradd -r -s /sbin/nologin baudlink

# Add to dialout group for serial access
usermod -aG dialout baudlink
```

**Windows:**

Use a managed service account or dedicated local account.

### 5. Logging and Monitoring

Enable comprehensive logging for audit trails:

```yaml
logging:
  level: "info"
  format: "json"
  file: "/var/log/baudlink/agent.log"
```

Monitor for:

- Unauthorized connection attempts
- Unusual port access patterns
- Service availability

## Threat Model

### Considered Threats

| Threat | Mitigation |
|--------|------------|
| Eavesdropping | TLS encryption |
| Port conflict | Exclusive locking |
| Unauthorized network access | Firewall rules, binding address |
| Configuration tampering | File permissions |
| Service compromise | Minimal privileges, service account |

### Out of Scope

- Physical security of hardware devices
- Host OS security hardening
- Application-layer authentication (implement in your client application)

## Security Reporting

If you discover a security vulnerability, please report it responsibly:

1. **Do not** create a public GitHub issue
2. Email security concerns to the project maintainers
3. Include detailed reproduction steps
4. Allow time for a fix before disclosure

## Changelog

- **v1.0.0** - Initial security model with TLS support
