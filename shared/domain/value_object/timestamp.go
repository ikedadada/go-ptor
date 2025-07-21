package value_object

import (
	"time"
)

type TimeStamp struct{ t time.Time }

// Now は現在時刻の TimeStamp を返します（UTC で固定）。
func Now() TimeStamp { return TimeStamp{time.Now().UTC()} }

// TimeStampFrom は外部 time.Time を UTC に正規化して保持します。
func TimeStampFrom(t time.Time) TimeStamp { return TimeStamp{t.UTC()} }

func (ts TimeStamp) Time() time.Time         { return ts.t }
func (ts TimeStamp) Unix() int64             { return ts.t.Unix() }
func (ts TimeStamp) String() string          { return ts.t.Format(time.RFC3339Nano) }
func (ts TimeStamp) Before(o TimeStamp) bool { return ts.t.Before(o.t) }
func (ts TimeStamp) After(o TimeStamp) bool  { return ts.t.After(o.t) }
func (ts TimeStamp) Equal(o TimeStamp) bool  { return ts.t.Equal(o.t) }
