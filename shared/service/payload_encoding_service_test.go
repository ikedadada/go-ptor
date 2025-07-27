package service

import (
	"testing"
)

func TestPayloadEncodingService_ExtendPayload(t *testing.T) {
	svc := NewPayloadEncodingService()

	original := &ExtendPayloadDTO{
		NextHop:   "127.0.0.1:5001",
		ClientPub: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
	}

	encoded, err := svc.EncodeExtendPayload(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := svc.DecodeExtendPayload(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.NextHop != original.NextHop {
		t.Errorf("NextHop mismatch: got %s, want %s", decoded.NextHop, original.NextHop)
	}
	if decoded.ClientPub != original.ClientPub {
		t.Errorf("ClientPub mismatch: got %v, want %v", decoded.ClientPub, original.ClientPub)
	}
}

func TestPayloadEncodingService_CreatedPayload(t *testing.T) {
	svc := NewPayloadEncodingService()

	original := &CreatedPayloadDTO{
		RelayPub: [32]byte{32, 31, 30, 29, 28, 27, 26, 25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
	}

	encoded, err := svc.EncodeCreatedPayload(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := svc.DecodeCreatedPayload(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.RelayPub != original.RelayPub {
		t.Errorf("RelayPub mismatch: got %v, want %v", decoded.RelayPub, original.RelayPub)
	}
}

func TestPayloadEncodingService_BeginPayload(t *testing.T) {
	svc := NewPayloadEncodingService()

	original := &BeginPayloadDTO{
		StreamID: 42,
		Target:   "example.com:80",
	}

	encoded, err := svc.EncodeBeginPayload(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := svc.DecodeBeginPayload(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.StreamID != original.StreamID {
		t.Errorf("StreamID mismatch: got %d, want %d", decoded.StreamID, original.StreamID)
	}
	if decoded.Target != original.Target {
		t.Errorf("Target mismatch: got %s, want %s", decoded.Target, original.Target)
	}
}

func TestPayloadEncodingService_ConnectPayload(t *testing.T) {
	svc := NewPayloadEncodingService()

	original := &ConnectPayloadDTO{
		Target: "hidden-service.onion:80",
	}

	encoded, err := svc.EncodeConnectPayload(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := svc.DecodeConnectPayload(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Target != original.Target {
		t.Errorf("Target mismatch: got %s, want %s", decoded.Target, original.Target)
	}
}

func TestPayloadEncodingService_DataPayload(t *testing.T) {
	svc := NewPayloadEncodingService()

	original := &DataPayloadDTO{
		StreamID: 123,
		Data:     []byte("Hello, world! This is test data."),
	}

	encoded, err := svc.EncodeDataPayload(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := svc.DecodeDataPayload(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.StreamID != original.StreamID {
		t.Errorf("StreamID mismatch: got %d, want %d", decoded.StreamID, original.StreamID)
	}
	if string(decoded.Data) != string(original.Data) {
		t.Errorf("Data mismatch: got %s, want %s", string(decoded.Data), string(original.Data))
	}
}

func TestPayloadEncodingService_EmptyData(t *testing.T) {
	svc := NewPayloadEncodingService()

	original := &DataPayloadDTO{
		StreamID: 1,
		Data:     []byte{},
	}

	encoded, err := svc.EncodeDataPayload(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := svc.DecodeDataPayload(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.StreamID != original.StreamID {
		t.Errorf("StreamID mismatch: got %d, want %d", decoded.StreamID, original.StreamID)
	}
	if len(decoded.Data) != 0 {
		t.Errorf("Expected empty data, got %v", decoded.Data)
	}
}

func TestPayloadEncodingService_InvalidData(t *testing.T) {
	svc := NewPayloadEncodingService()

	// Test with corrupted data
	invalidData := []byte{0xFF, 0xFF, 0xFF, 0xFF}

	_, err := svc.DecodeDataPayload(invalidData)
	if err == nil {
		t.Error("Expected decode to fail with invalid data")
	}

	_, err = svc.DecodeBeginPayload(invalidData)
	if err == nil {
		t.Error("Expected decode to fail with invalid data")
	}

	_, err = svc.DecodeExtendPayload(invalidData)
	if err == nil {
		t.Error("Expected decode to fail with invalid data")
	}
}
