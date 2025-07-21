package value_object_test

import (
	vo "ikedadada/go-ptor/shared/domain/value_object"
	"testing"
	"time"
)

func TestTimeStamp_Table(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	tests := []struct {
		name        string
		input       time.Time
		expectsUnix int64
	}{
		{"now", now, now.Unix()},
		{"past", past, past.Unix()},
		{"future", future, future.Unix()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := vo.TimeStampFrom(tt.input)
			if ts.Unix() != tt.expectsUnix {
				t.Errorf("expected unix %d, got %d", tt.expectsUnix, ts.Unix())
			}
		})
	}
}
