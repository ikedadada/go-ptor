package value_object

import "testing"

func TestDataPayload_RoundTrip(t *testing.T) {
	p := &DataPayload{StreamID: 2, Data: []byte("hi")}
	b, err := EncodeDataPayload(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out, err := DecodeDataPayload(b)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.StreamID != p.StreamID || string(out.Data) != string(p.Data) {
		t.Errorf("round-trip mismatch")
	}
}
