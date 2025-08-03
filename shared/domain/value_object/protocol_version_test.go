package value_object

import (
	"testing"
)

func TestProtocolVersion_String(t *testing.T) {
	tests := []struct {
		name     string
		version  ProtocolVersion
		expected string
	}{
		{"ProtocolV1", ProtocolV1, "v1"},
		{"Version_2", ProtocolVersion(0x02), "unknown(2)"},
		{"Version_0", ProtocolVersion(0x00), "unknown(0)"},
		{"Version_255", ProtocolVersion(0xFF), "unknown(255)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestProtocolVersion_IsSupported(t *testing.T) {
	tests := []struct {
		name      string
		version   ProtocolVersion
		supported bool
	}{
		{"ProtocolV1", ProtocolV1, true},
		{"Version_2", ProtocolVersion(0x02), false},
		{"Version_0", ProtocolVersion(0x00), false},
		{"Version_255", ProtocolVersion(0xFF), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.IsSupported()
			if result != tt.supported {
				t.Errorf("IsSupported() = %v, want %v", result, tt.supported)
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

func TestProtocolVersion_AllVersions(t *testing.T) {
	tests := []struct {
		name            string
		version         ProtocolVersion
		expectSupported bool
	}{
		{"ProtocolV1", ProtocolV1, true},
		{"Zero value", ProtocolVersion(0x00), false},
		{"Future version", ProtocolVersion(0x02), false},
		{"Random version", ProtocolVersion(0x10), false},
		{"Maximum byte value", ProtocolVersion(0xFF), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSupported := tt.version.IsSupported()
			if isSupported != tt.expectSupported {
				t.Errorf("Protocol version %v IsSupported() = %v, want %v", tt.version, isSupported, tt.expectSupported)
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.version.IsSupported() != tt.expectSupported {
				t.Errorf("IsSupported() = %v, want %v", tt.version.IsSupported(), tt.expectSupported)
			}
			if tt.version.String() != tt.expectString {
				t.Errorf("String() = %q, want %q", tt.version.String(), tt.expectString)
			}
		})
	}
}
