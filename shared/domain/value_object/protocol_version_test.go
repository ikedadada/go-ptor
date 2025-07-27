package value_object

import (
	"testing"
)

func TestProtocolVersion_String(t *testing.T) {
	tests := []struct {
		version  ProtocolVersion
		expected string
	}{
		{ProtocolV1, "v1"},
		{ProtocolVersion(0x02), "unknown(2)"},
		{ProtocolVersion(0x00), "unknown(0)"},
		{ProtocolVersion(0xFF), "unknown(255)"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.version.String()
			if result != test.expected {
				t.Errorf("String() = %q, want %q", result, test.expected)
			}
		})
	}
}

func TestProtocolVersion_IsSupported(t *testing.T) {
	tests := []struct {
		version   ProtocolVersion
		supported bool
	}{
		{ProtocolV1, true},
		{ProtocolVersion(0x02), false},
		{ProtocolVersion(0x00), false},
		{ProtocolVersion(0xFF), false},
	}

	for _, test := range tests {
		t.Run(test.version.String(), func(t *testing.T) {
			result := test.version.IsSupported()
			if result != test.supported {
				t.Errorf("IsSupported() = %v, want %v", result, test.supported)
			}
		})
	}
}

func TestProtocolVersion_Constants(t *testing.T) {
	// Test that constants have expected values
	if ProtocolV1 != 0x01 {
		t.Errorf("ProtocolV1 = %d, want %d", ProtocolV1, 0x01)
	}
}

func TestProtocolVersion_ByteConversion(t *testing.T) {
	// Test that ProtocolVersion can be converted to/from byte
	var version ProtocolVersion = ProtocolV1
	byteValue := byte(version)
	if byteValue != 0x01 {
		t.Errorf("byte(ProtocolV1) = %d, want %d", byteValue, 0x01)
	}

	// Test conversion from byte
	convertedVersion := ProtocolVersion(byteValue)
	if convertedVersion != ProtocolV1 {
		t.Errorf("ProtocolVersion(%d) = %v, want %v", byteValue, convertedVersion, ProtocolV1)
	}
}

func TestProtocolVersion_AllSupportedVersions(t *testing.T) {
	tests := []struct {
		name    string
		version ProtocolVersion
	}{
		{"ProtocolV1", ProtocolV1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !test.version.IsSupported() {
				t.Errorf("Protocol version %v should be supported", test.version)
			}
		})
	}
}

func TestProtocolVersion_UnsupportedVersions(t *testing.T) {
	tests := []struct {
		name    string
		version ProtocolVersion
	}{
		{"Zero value", ProtocolVersion(0x00)},
		{"Future version", ProtocolVersion(0x02)},
		{"Random version", ProtocolVersion(0x10)},
		{"Maximum byte value", ProtocolVersion(0xFF)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.version.IsSupported() {
				t.Errorf("Protocol version %v should not be supported", test.version)
			}
		})
	}
}

func TestProtocolVersion_StringConsistency(t *testing.T) {
	// Test that String() method is consistent
	version := ProtocolV1
	str1 := version.String()
	str2 := version.String()

	if str1 != str2 {
		t.Errorf("String() method should be consistent: %q != %q", str1, str2)
	}

	if str1 != "v1" {
		t.Errorf("String() should return 'v1' for ProtocolV1, got %q", str1)
	}
}

func TestProtocolVersion_Zero(t *testing.T) {
	// Test zero value behavior
	var version ProtocolVersion

	if version.IsSupported() {
		t.Error("Zero value ProtocolVersion should not be supported")
	}

	expected := "unknown(0)"
	if version.String() != expected {
		t.Errorf("Zero value String() = %q, want %q", version.String(), expected)
	}
}

func TestProtocolVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		version         ProtocolVersion
		expectSupported bool
		expectString    string
	}{
		{"Same as ProtocolV1", ProtocolVersion(1), true, "v1"},
		{"Value 127", ProtocolVersion(127), false, "unknown(127)"},
		{"Value 128", ProtocolVersion(128), false, "unknown(128)"},
		{"Value 254", ProtocolVersion(254), false, "unknown(254)"},
		{"Value 255", ProtocolVersion(255), false, "unknown(255)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.version.IsSupported() != test.expectSupported {
				t.Errorf("IsSupported() = %v, want %v", test.version.IsSupported(), test.expectSupported)
			}
			if test.version.String() != test.expectString {
				t.Errorf("String() = %q, want %q", test.version.String(), test.expectString)
			}
		})
	}
}
