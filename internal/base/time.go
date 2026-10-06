package base

import "time"

// MicroTime stores time in unix microseconds timestamp.
type MicroTime uint64

func (t MicroTime) String() string {
	if t == 0 {
		return ""
	}
	return time.UnixMicro(int64(t)).Format(time.RFC3339Nano)
}

func (t MicroTime) Time() time.Time {
	if t == 0 {
		return time.Time{}
	}
	return time.UnixMicro(int64(t))
}

func (t MicroTime) Add(d time.Duration) MicroTime {
	return t + MicroTime(d.Microseconds())
}

func (t MicroTime) Raw() uint64 {
	return uint64(t)
}

func Now() MicroTime {
	return MicroTime(time.Now().UnixMicro())
}

func FromNow(d time.Duration) MicroTime {
	return MicroTime(time.Now().Add(d).UnixMicro())
}
