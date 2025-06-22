package value_object

import "testing"

func TestExtendPayload_RoundTrip(t *testing.T) {
	p := &ExtendPayload{NextHop: "127.0.0.1:5001", EncKey: []byte("secret")}
	b, err := EncodeExtendPayload(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out, err := DecodeExtendPayload(b)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.NextHop != p.NextHop || string(out.EncKey) != string(p.EncKey) {
		t.Errorf("mismatch: %+v vs %+v", out, p)
	}
}
