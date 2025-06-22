package value_object

import "testing"

func TestExtendPayload_RoundTrip(t *testing.T) {
	var pub [32]byte
	copy(pub[:], []byte("0123456789abcdef0123456789abcdef"))
	p := &ExtendPayload{NextHop: "127.0.0.1:5001", ClientPub: pub}
	b, err := EncodeExtendPayload(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out, err := DecodeExtendPayload(b)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.NextHop != p.NextHop || out.ClientPub != p.ClientPub {
		t.Errorf("mismatch: %+v vs %+v", out, p)
	}
}
