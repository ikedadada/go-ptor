package service

import (
	"bytes"
	"encoding/gob"
)

// PayloadEncodingService handles encoding and decoding of cell payloads
type PayloadEncodingService interface {
	EncodeExtendPayload(*ExtendPayloadDTO) ([]byte, error)
	DecodeExtendPayload([]byte) (*ExtendPayloadDTO, error)
	EncodeCreatedPayload(*CreatedPayloadDTO) ([]byte, error)
	DecodeCreatedPayload([]byte) (*CreatedPayloadDTO, error)
	EncodeBeginPayload(*BeginPayloadDTO) ([]byte, error)
	DecodeBeginPayload([]byte) (*BeginPayloadDTO, error)
	EncodeConnectPayload(*ConnectPayloadDTO) ([]byte, error)
	DecodeConnectPayload([]byte) (*ConnectPayloadDTO, error)
	EncodeDataPayload(*DataPayloadDTO) ([]byte, error)
	DecodeDataPayload([]byte) (*DataPayloadDTO, error)
}

// ExtendPayloadDTO carries the information needed to extend a circuit to the next hop.
type ExtendPayloadDTO struct {
	NextHop   string
	ClientPub [32]byte
}

// CreatedPayloadDTO carries the relay's public key for a new circuit hop.
type CreatedPayloadDTO struct {
	RelayPub [32]byte
}

// BeginPayloadDTO specifies the target address for a new stream.
type BeginPayloadDTO struct {
	StreamID uint16
	Target   string
}

// ConnectPayloadDTO specifies the hidden service address for CONNECT command.
type ConnectPayloadDTO struct {
	Target string
}

// DataPayloadDTO represents application data flowing through a circuit.
type DataPayloadDTO struct {
	StreamID uint16
	Data     []byte
}

type payloadEncodingServiceImpl struct{}

// NewPayloadEncodingService creates a new payload encoding service
func NewPayloadEncodingService() PayloadEncodingService {
	return &payloadEncodingServiceImpl{}
}

func (s *payloadEncodingServiceImpl) EncodeExtendPayload(p *ExtendPayloadDTO) ([]byte, error) {
	return encodePayload(p)
}

func (s *payloadEncodingServiceImpl) DecodeExtendPayload(data []byte) (*ExtendPayloadDTO, error) {
	return decodePayload[ExtendPayloadDTO](data)
}

func (s *payloadEncodingServiceImpl) EncodeCreatedPayload(p *CreatedPayloadDTO) ([]byte, error) {
	return encodePayload(p)
}

func (s *payloadEncodingServiceImpl) DecodeCreatedPayload(data []byte) (*CreatedPayloadDTO, error) {
	return decodePayload[CreatedPayloadDTO](data)
}

func (s *payloadEncodingServiceImpl) EncodeBeginPayload(p *BeginPayloadDTO) ([]byte, error) {
	return encodePayload(p)
}

func (s *payloadEncodingServiceImpl) DecodeBeginPayload(data []byte) (*BeginPayloadDTO, error) {
	return decodePayload[BeginPayloadDTO](data)
}

func (s *payloadEncodingServiceImpl) EncodeConnectPayload(p *ConnectPayloadDTO) ([]byte, error) {
	return encodePayload(p)
}

func (s *payloadEncodingServiceImpl) DecodeConnectPayload(data []byte) (*ConnectPayloadDTO, error) {
	return decodePayload[ConnectPayloadDTO](data)
}

func (s *payloadEncodingServiceImpl) EncodeDataPayload(p *DataPayloadDTO) ([]byte, error) {
	return encodePayload(p)
}

func (s *payloadEncodingServiceImpl) DecodeDataPayload(data []byte) (*DataPayloadDTO, error) {
	return decodePayload[DataPayloadDTO](data)
}

// encodePayload serializes any type using gob encoding
func encodePayload(payload any) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(payload)
	return buf.Bytes(), err
}

// decodePayload deserializes bytes into the specified type using generics
func decodePayload[T any](data []byte) (*T, error) {
	var result T
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&result)
	return &result, err
}
