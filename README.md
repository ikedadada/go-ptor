# go-ptor

This repository demonstrates a simple multi-layer Go project. Code is organized following the layered layout described in `docs/style_backend.md`:

```
internal/
  domain/        # core models and business logic
  usecase/       # application orchestration logic
  handler/       # HTTP/CLI adapters (thin layer)
  infrastructure/ # external systems and drivers
```

The separation keeps business rules isolated from entry points and infrastructure code.

## Requirements

- Go **1.24** or higher (see `mise.toml`)

## Setup

1. Install Go and ensure `go` is in your `PATH`.
2. Clone this repository and download dependencies:

   ```bash
   go mod download
   ```

## Usage

Each entry in `cmd/` is a standalone binary. You can run them directly with `go run` or build them using `go build`.
For example, to start the client:

```bash
go run ./cmd/client
```

When the client resolves a `.ptor` address, it consults the directory to find
the designated exit relay for that service. A new circuit is built so this relay
is the final hop. The client then sends a CONNECT cell once for that circuit
before any streams are opened. After the CONNECT succeeds, stream data continues
to use BEGIN cells as before. The CONNECT payload is onion encrypted across all
hops so only the exit relay can read it.

### Environment variables

The relay uses the `PTOR_HIDDEN_ADDR` variable to locate the hidden HTTP service
when processing CONNECT cells. If not set, it falls back to `hidden:5000` (the
Docker demo value). The older `HIDDEN_ADDR` variable is also checked for
backward compatibility.

Relay connections are kept in an in-memory table. The TTL can be configured
with the `-ttl` flag when starting the relay. Increase this value when
troubleshooting long-lived circuits. A default TTL can also be provided via the
`PTOR_TTL_SECONDS` environment variable which specifies the number of seconds
to keep each circuit entry.

Additional logging now records failures when decoding EXTEND payloads,
connecting to next hops, or forwarding cells so issues can be diagnosed more
easily.

### Hidden service

The hidden service proxies incoming connections to an upstream HTTP server. Use
the `-http` flag to specify the target address.

The provided `docker compose` configuration starts a small demo server from
`cmd/httpdemo` and points the hidden service at `httpdemo:8080`. Bring up the
demo stack with:

```bash
docker compose up --build
```

The hidden service will print its `.ptor` address on startup. You can access it
via the client once the stack is running.

## Testing

Execute all unit tests with:

```bash
go test ./...
```

## TODO: Tor Protocol Compatibility

This implementation is currently an educational demonstration of onion routing concepts and is **completely incompatible** with the actual Tor network. The following changes would be needed to achieve Tor protocol compatibility:

### Phase 1: Core Protocol Compatibility (High Priority)

#### 1. Cell Structure Redesign
- [ ] **Replace custom cell format with standard 512-byte fixed cells**
  - Current: Custom `Cmd|Version|Length|Payload|Padding` format
  - Target: Fixed 512-byte cells with proper Tor cell structure
  - Files: `internal/domain/value_object/cell.go`, `internal/handler/`

#### 2. RELAY Cell Implementation
- [ ] **Wrap all relay commands (BEGIN/DATA/END) in RELAY cells**
  - Current: Direct command processing in relays
  - Target: All application-level commands wrapped in RELAY cells
  - Impact: Complete relay processing logic rewrite
  - Files: `internal/usecase/relay_usecase.go`, `internal/usecase/send_data_usecase.go`

#### 3. Encryption Algorithm Migration
- [ ] **Replace AES-256-GCM with AES-128-CTR**
  - Current: AES-GCM with custom nonce management
  - Target: AES-CTR as per Tor specification
  - Files: `internal/infrastructure/service/crypto_service.go`
  - Impact: Complete cryptographic rewrite, key derivation changes

#### 4. Remove Custom CONNECT Command
- [ ] **Eliminate CmdConnect and use standard RELAY_BEGIN**
  - Current: Custom CONNECT command for hidden services
  - Target: Standard Tor RELAY_BEGIN command only
  - Files: `internal/usecase/connect_usecase.go`, `internal/usecase/relay_usecase.go`
  - Impact: Hidden service connection logic rewrite

### Phase 2: Protocol Adherence (Medium Priority)

#### 5. Circuit Building Protocol
- [ ] **Implement TAP/ntor handshake protocols**
  - Current: X25519 ECDH + HKDF
  - Target: Standard Tor TAP or ntor protocol
  - Files: `internal/usecase/service/circuit_build_service.go`
  - Impact: Complete circuit establishment rewrite

#### 6. Flow Control Implementation
- [ ] **Add SENDME cells for congestion control**
  - Current: No flow control mechanism
  - Target: SENDME-based flow control per Tor spec
  - Files: New files needed, relay processing updates

#### 7. Standard Tor Commands
- [ ] **Implement missing Tor commands**
  - Add: PADDING, RELAY_EARLY, CREATE_FAST, CREATED_FAST
  - Add: VERSIONS, NETINFO, RELAY_COMMAND_* variants
  - Files: `internal/domain/value_object/cell.go`, relay processing

#### 8. Proper Relay Forwarding
- [ ] **Fix middle relay processing to maintain end-to-end encryption**
  - Current: Middle relays decrypt application data
  - Target: Layer-by-layer onion peeling only
  - Files: `internal/usecase/relay_usecase.go`
  - Impact: Fundamental relay processing changes

### Phase 3: Complete Tor Compatibility (Lower Priority)

#### 9. Directory System Overhaul
- [ ] **Replace HTTP directory with Tor directory protocol**
  - Current: Centralized HTTP-based directory service
  - Target: Distributed directory authorities with consensus
  - Files: Complete directory service rewrite
  - Impact: New consensus protocol, descriptor parsing

#### 10. Hidden Service Protocol v3
- [ ] **Implement proper hidden service protocol**
  - Current: Custom .ptor addresses with direct connections
  - Target: .onion v3 addresses with introduction/rendezvous points
  - Files: Complete hidden service rewrite
  - Impact: Introduction points, rendezvous protocol, descriptor publication

#### 11. Guard Relay System
- [ ] **Add entry guard selection and management**
  - Current: Random relay selection per circuit
  - Target: Persistent guard relay selection
  - Files: New guard management system
  - Impact: Client-side guard relay logic

#### 12. Advanced Features
- [ ] **Bandwidth measurement and weighted relay selection**
- [ ] **Proper circuit multiplexing and load balancing**
- [ ] **Bridge support for censorship circumvention**
- [ ] **Pluggable transports interface**

### Technical Debt and Code Quality

#### 13. Architecture Improvements
- [ ] **Add comprehensive error handling and recovery**
- [ ] **Implement proper logging with log levels**
- [ ] **Add configuration management system**
- [ ] **Performance optimization for cell processing**

#### 14. Testing and Validation
- [ ] **Add Tor protocol compliance tests**
- [ ] **Implement integration tests with real Tor clients**
- [ ] **Add performance benchmarks**
- [ ] **Security audit and vulnerability assessment**

### Current Status

**⚠️ Warning**: This implementation is for educational purposes only and should never be used for actual anonymity or privacy protection. Key differences from Tor:

- **Incompatible cell format**: Custom commands and structure
- **Wrong encryption**: AES-GCM instead of AES-CTR
- **Missing RELAY cells**: Direct command processing breaks onion routing
- **Custom hidden services**: .ptor addresses instead of .onion
- **No flow control**: Missing SENDME mechanism
- **Centralized directory**: HTTP-based instead of consensus
- **Simplified handshakes**: X25519 instead of TAP/ntor

### Contributing

When working on Tor compatibility, please:
1. Reference the [Tor Protocol Specification](https://spec.torproject.org/)
2. Test changes against the official Tor implementation
3. Maintain backward compatibility for educational demos
4. Update this TODO list as items are completed

For questions about Tor protocol details, consult:
- [Tor Specification](https://spec.torproject.org/)
- [Tor Research](https://research.torproject.org/)
- [Tor Source Code](https://gitlab.torproject.org/tpo/core/tor)
