package service

import (
	"fmt"
	
	"ikedadada/go-ptor/internal/domain/entity"
	"ikedadada/go-ptor/internal/domain/value_object"
	"ikedadada/go-ptor/internal/usecase/service"
)

// circuitCryptographyServiceImpl implements CircuitCryptographyService
type circuitCryptographyServiceImpl struct {
	cryptoService service.CryptoService
}

// NewCircuitCryptographyService creates a new CircuitCryptographyService
func NewCircuitCryptographyService(cryptoService service.CryptoService) CircuitCryptographyService {
	return &circuitCryptographyServiceImpl{
		cryptoService: cryptoService,
	}
}

// EncryptForCircuit encrypts payload for multi-hop circuit transmission
func (s *circuitCryptographyServiceImpl) EncryptForCircuit(circuit *entity.Circuit, messageType entity.MessageType, payload []byte) ([]byte, error) {
	keys := make([][32]byte, 0, len(circuit.Hops()))
	nonces := make([][12]byte, 0, len(circuit.Hops()))
	
	// Generate nonces for each hop in forward order
	for i := range circuit.Hops() {
		keys = append(keys, circuit.HopKey(i))
		nonces = append(nonces, s.getNonceForCircuitHop(circuit, i, messageType))
	}
	
	return s.cryptoService.AESMultiSeal(keys, nonces, payload)
}

// DecryptAtRelay decrypts one layer of encryption at a relay
func (s *circuitCryptographyServiceImpl) DecryptAtRelay(connState *entity.ConnState, messageType entity.MessageType, payload []byte) ([]byte, bool, error) {
	nonce := s.getNonceForRelay(connState, messageType)
	
	decrypted, err := s.cryptoService.AESOpen(connState.Key(), nonce, payload)
	if err != nil {
		// Decryption failure might indicate upstream data flow
		return nil, false, err
	}
	
	return decrypted, true, nil
}

// EncryptAtRelay adds one layer of encryption at a relay for upstream data
func (s *circuitCryptographyServiceImpl) EncryptAtRelay(connState *entity.ConnState, messageType entity.MessageType, payload []byte) ([]byte, error) {
	nonce := s.getNonceForRelayUpstream(connState, messageType)
	return s.cryptoService.AESSeal(connState.Key(), nonce, payload)
}

// DecryptMultiLayer performs multi-layer decryption for client
func (s *circuitCryptographyServiceImpl) DecryptMultiLayer(circuit *entity.Circuit, messageType entity.MessageType, payload []byte) ([]byte, error) {
	data := payload
	hopCount := len(circuit.Hops())
	
	// Decrypt each layer in circuit order (first hop to exit hop)
	for hop := 0; hop < hopCount; hop++ {
		key := circuit.HopKey(hop)
		nonce := s.getNonceForCircuitHopUpstream(circuit, hop, messageType)
		
		decrypted, err := s.cryptoService.AESOpen(key, nonce, data)
		if err != nil {
			return nil, fmt.Errorf("decrypt layer %d failed: %w", hop, err)
		}
		data = decrypted
	}
	
	return data, nil
}

// GenerateNonceForHop generates the appropriate nonce for a specific hop and message type
func (s *circuitCryptographyServiceImpl) GenerateNonceForHop(circuit *entity.Circuit, hopIndex int, messageType entity.MessageType) value_object.Nonce {
	return s.getNonceForCircuitHop(circuit, hopIndex, messageType)
}

// GenerateNonceForRelay generates the appropriate nonce for relay operations
func (s *circuitCryptographyServiceImpl) GenerateNonceForRelay(connState *entity.ConnState, messageType entity.MessageType) value_object.Nonce {
	return s.getNonceForRelay(connState, messageType)
}

// AdvanceNonce increments the nonce counter for the given message type
func (s *circuitCryptographyServiceImpl) AdvanceNonce(entity NonceCounterEntity, messageType entity.MessageType) {
	entity.IncrementCounter(messageType)
}

// getNonceForCircuitHop gets the appropriate nonce for circuit hop (downstream)
func (s *circuitCryptographyServiceImpl) getNonceForCircuitHop(circuit *entity.Circuit, hopIndex int, messageType entity.MessageType) value_object.Nonce {
	switch messageType {
	case entity.MessageTypeBegin, entity.MessageTypeConnect:
		return circuit.HopBeginNonce(hopIndex)
	case entity.MessageTypeData:
		return circuit.HopDataNonce(hopIndex)
	default:
		return circuit.HopDataNonce(hopIndex)
	}
}

// getNonceForCircuitHopUpstream gets the appropriate nonce for circuit hop (upstream)
func (s *circuitCryptographyServiceImpl) getNonceForCircuitHopUpstream(circuit *entity.Circuit, hopIndex int, messageType entity.MessageType) value_object.Nonce {
	switch messageType {
	case entity.MessageTypeUpstreamData:
		return circuit.HopUpstreamDataNonce(hopIndex)
	default:
		return circuit.HopUpstreamDataNonce(hopIndex)
	}
}

// getNonceForRelay gets the appropriate nonce for relay operations (downstream)
func (s *circuitCryptographyServiceImpl) getNonceForRelay(connState *entity.ConnState, messageType entity.MessageType) value_object.Nonce {
	switch messageType {
	case entity.MessageTypeBegin, entity.MessageTypeConnect:
		return connState.BeginNonce()
	case entity.MessageTypeData:
		return connState.DataNonce()
	default:
		return connState.DataNonce()
	}
}

// getNonceForRelayUpstream gets the appropriate nonce for relay operations (upstream)
func (s *circuitCryptographyServiceImpl) getNonceForRelayUpstream(connState *entity.ConnState, messageType entity.MessageType) value_object.Nonce {
	switch messageType {
	case entity.MessageTypeUpstreamData:
		return connState.UpstreamDataNonce()
	default:
		return connState.UpstreamDataNonce()
	}
}