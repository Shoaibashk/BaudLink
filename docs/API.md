# API Documentation

BaudLink exposes a gRPC API for serial port management. This document covers all available RPC methods and message types.

## Quick Start

### Connecting to the Service

**Python:**

```python
import grpc
from serial_pb2_grpc import SerialServiceStub

# Without TLS
channel = grpc.insecure_channel('localhost:50051')

# With TLS
# creds = grpc.ssl_channel_credentials(open('cert.pem', 'rb').read())
# channel = grpc.secure_channel('localhost:50051', creds)

stub = SerialServiceStub(channel)
```

**Go:**

```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "github.com/Shoaibashk/BaudLink/api/proto"
)

// Without TLS
conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))

// With TLS
// creds, _ := credentials.NewClientTLSFromFile("cert.pem", "")
// conn, _ := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(creds))

client := pb.NewSerialServiceClient(conn)
```

## Service Definition

```protobuf
service SerialService {
  // Port discovery and management
  rpc ListPorts(ListPortsRequest) returns (ListPortsResponse);
  rpc OpenPort(OpenPortRequest) returns (OpenPortResponse);
  rpc ClosePort(ClosePortRequest) returns (ClosePortResponse);
  
  // Data operations
  rpc Write(WriteRequest) returns (WriteResponse);
  rpc Read(ReadRequest) returns (ReadResponse);
  
  // Streaming operations
  rpc StreamRead(StreamReadRequest) returns (stream ReadData);
  rpc StreamWrite(stream WriteData) returns (StreamWriteResponse);
  rpc BiDirectionalStream(stream WriteData) returns (stream ReadData);
  
  // Agent information
  rpc GetAgentInfo(GetAgentInfoRequest) returns (AgentInfo);
}
```

## RPC Methods

### ListPorts

Discover all available serial ports on the system.

**Request:** `ListPortsRequest` (empty message)

**Response:** `ListPortsResponse`

| Field | Type | Description |
|-------|------|-------------|
| ports | repeated PortInfo | List of discovered ports |

**Example:**

```python
response = stub.ListPorts(ListPortsRequest())
for port in response.ports:
    print(f"{port.name}: {port.description}")
    print(f"  VID:PID = {port.vid:04X}:{port.pid:04X}")
```

---

### OpenPort

Open a serial port with specified configuration.

**Request:** `OpenPortRequest`

| Field | Type | Description |
|-------|------|-------------|
| port_name | string | Port name (e.g., "COM3", "/dev/ttyUSB0") |
| config | PortConfig | Port configuration |

**PortConfig Fields:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| baud_rate | int32 | 9600 | Baud rate |
| data_bits | int32 | 8 | Data bits (5, 6, 7, 8) |
| stop_bits | StopBits | ONE | Stop bits (ONE, ONE_HALF, TWO) |
| parity | Parity | NONE | Parity (NONE, ODD, EVEN, MARK, SPACE) |
| read_timeout_ms | int32 | 0 | Read timeout in milliseconds (0 = blocking) |

**Response:** `OpenPortResponse`

| Field | Type | Description |
|-------|------|-------------|
| success | bool | Whether the port was opened |
| port_handle | string | Handle for subsequent operations |
| error | string | Error message if failed |

**Example:**

```python
config = PortConfig(
    baud_rate=115200,
    data_bits=8,
    stop_bits=StopBits.ONE,
    parity=Parity.NONE,
    read_timeout_ms=1000
)

response = stub.OpenPort(OpenPortRequest(
    port_name="COM3",
    config=config
))

if response.success:
    handle = response.port_handle
    print(f"Port opened: {handle}")
```

---

### ClosePort

Close an open serial port.

**Request:** `ClosePortRequest`

| Field | Type | Description |
|-------|------|-------------|
| port_handle | string | Handle from OpenPort |

**Response:** `ClosePortResponse`

| Field | Type | Description |
|-------|------|-------------|
| success | bool | Whether the port was closed |
| error | string | Error message if failed |

---

### Write

Write data to an open port.

**Request:** `WriteRequest`

| Field | Type | Description |
|-------|------|-------------|
| port_handle | string | Handle from OpenPort |
| data | bytes | Data to write |

**Response:** `WriteResponse`

| Field | Type | Description |
|-------|------|-------------|
| bytes_written | int32 | Number of bytes written |
| success | bool | Whether write succeeded |
| error | string | Error message if failed |

**Example:**

```python
response = stub.Write(WriteRequest(
    port_handle=handle,
    data=b"Hello, Device!\n"
))
print(f"Wrote {response.bytes_written} bytes")
```

---

### Read

Read data from an open port.

**Request:** `ReadRequest`

| Field | Type | Description |
|-------|------|-------------|
| port_handle | string | Handle from OpenPort |
| max_bytes | int32 | Maximum bytes to read |
| timeout_ms | int32 | Read timeout (0 = use port default) |

**Response:** `ReadResponse`

| Field | Type | Description |
|-------|------|-------------|
| data | bytes | Data read from port |
| bytes_read | int32 | Number of bytes read |
| success | bool | Whether read succeeded |
| error | string | Error message if failed |

---

### StreamRead

Stream data continuously from a port.

**Request:** `StreamReadRequest`

| Field | Type | Description |
|-------|------|-------------|
| port_handle | string | Handle from OpenPort |
| buffer_size | int32 | Read buffer size |

**Response:** Stream of `ReadData`

| Field | Type | Description |
|-------|------|-------------|
| data | bytes | Chunk of received data |
| timestamp | int64 | Unix timestamp (nanoseconds) |

**Example:**

```python
for chunk in stub.StreamRead(StreamReadRequest(
    port_handle=handle,
    buffer_size=256
)):
    print(f"Received: {chunk.data}")
```

---

### StreamWrite

Stream data to a port.

**Request:** Stream of `WriteData`

| Field | Type | Description |
|-------|------|-------------|
| port_handle | string | Handle from OpenPort |
| data | bytes | Chunk of data to write |

**Response:** `StreamWriteResponse`

| Field | Type | Description |
|-------|------|-------------|
| total_bytes_written | int64 | Total bytes written |
| success | bool | Whether all writes succeeded |
| error | string | Error message if failed |

---

### BiDirectionalStream

Full-duplex bidirectional streaming.

**Request:** Stream of `WriteData`
**Response:** Stream of `ReadData`

**Example:**

```python
def generate_commands():
    for cmd in ["AT\r\n", "AT+VERSION\r\n", "AT+NAME?\r\n"]:
        yield WriteData(port_handle=handle, data=cmd.encode())

for response in stub.BiDirectionalStream(generate_commands()):
    print(f"Response: {response.data.decode()}")
```

---

### GetAgentInfo

Get information about the BaudLink agent.

**Request:** `GetAgentInfoRequest` (empty message)

**Response:** `AgentInfo`

| Field | Type | Description |
|-------|------|-------------|
| version | string | Agent version |
| platform | string | Operating system |
| uptime_seconds | int64 | Time since agent started |
| open_ports | int32 | Number of currently open ports |
| features | repeated string | Supported features |

## Message Types

### PortInfo

Information about a discovered serial port.

| Field | Type | Description |
|-------|------|-------------|
| name | string | Port name (COM3, /dev/ttyUSB0) |
| description | string | Human-readable description |
| hardware_id | string | Hardware identifier |
| vid | int32 | USB Vendor ID |
| pid | int32 | USB Product ID |
| serial_number | string | USB serial number |
| is_usb | bool | Whether it's a USB port |
| manufacturer | string | Device manufacturer |
| product | string | Product name |

### Enumerations

**StopBits:**

- `STOP_BITS_UNSPECIFIED` (0)
- `ONE` (1)
- `ONE_HALF` (2)
- `TWO` (3)

**Parity:**

- `PARITY_UNSPECIFIED` (0)
- `NONE` (1)
- `ODD` (2)
- `EVEN` (3)
- `MARK` (4)
- `SPACE` (5)

## Error Handling

All responses include error information:

```python
response = stub.OpenPort(request)
if not response.success:
    print(f"Error: {response.error}")
```

Common errors:

| Error | Description |
|-------|-------------|
| "port not found" | Specified port doesn't exist |
| "port already open" | Port is in use |
| "permission denied" | Insufficient permissions |
| "invalid configuration" | Bad port configuration |
| "port not open" | Operating on closed port |
| "write timeout" | Write operation timed out |
| "read timeout" | Read operation timed out |

## Client Libraries

Generate client code from the proto file:

```bash
# Python
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. serial.proto

# Go
protoc --go_out=. --go-grpc_out=. serial.proto

# C#
protoc --csharp_out=. --grpc_csharp_out=. serial.proto

# Node.js
grpc_tools_node_protoc --js_out=. --grpc_out=. serial.proto
```

## Rate Limiting

The server limits connections per client:

- Maximum 100 concurrent connections (configurable)
- Per-port exclusive access (one client per port)

## Best Practices

1. **Always close ports** when done to release resources
2. **Use streaming** for continuous data transfer
3. **Handle disconnections** gracefully in streaming clients
4. **Set appropriate timeouts** based on your device's response time
5. **Enable TLS** for production deployments over networks
