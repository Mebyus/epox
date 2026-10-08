package base

import (
	"encoding/json"
	"strconv"
	"time"
)

type RepTaskID uint64

func (i RepTaskID) String() string {
	if i == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(i), 10)
}

// RepTask represents a repeatable task which creates
// new tasks automatically based on configured schedule.
type RepTask struct {
	Tags []Tag

	Schedule RepSchedule

	Title       string
	Description string

	NextTrigger MicroTime

	// Zero value means no limit.
	TimeLimit time.Duration

	CreateTime MicroTime
	UpdateTime MicroTime

	ID     RepTaskID
	UserID UserID

	Timezone *time.Location

	// Higher values mean higher importance.
	Importance uint32

	// Equals zero if task does not have progress indicator.
	MaxProgress uint32
}

// RepTaskQueueEntry small structure used for initial loading of
// all repeatable tasks when server starts.
type RepTaskQueueEntry struct {
	ID RepTaskID

	NextTrigger MicroTime
}

// RepSchedule determines a schedule for triggering
// repeatable tasks.
type RepSchedule interface {
	Type() RepTaskType

	// Returns time of next trigger based on current time.
	// Implementations must respect timezone in supplied time.
	Next(now time.Time) time.Time

	// Reports in what timezone schedule operates.
	Timezone() *time.Location

	// Serialize itself into format compatible with Unmarshal.
	Marshal() ([]byte, time.Time, error)

	// Restore itself from serialized data provided by Marshal.
	Unmarshal(data []byte, next time.Time) error
}

type RepTaskType uint32

const (
	RepTaskInterval RepTaskType = iota
	RepTaskDaily
	RepTaskWeekly
	RepTaskMontly
	RepTaskYearly
)

type DailySchedule struct {
	// time of previous cached trigger
	//
	// zero means that it should be recalculated
	//
	// we cache it because the common usage for schedule is
	// to trigger it consequently
	prev MicroTime

	tz *time.Location

	// number of seconds from day start (with respect to timezone)
	// when trigger should occur
	local uint32

	// number of additional days to delay between triggers
	//
	// zero means that trigger will occur every day
	daydelay uint32
}

// NewDailySchedule creates daily schedule object with specified
// previous trigger point and additional delay (in days) between
// consecutive triggers.
//
// Zero value of daydelay means that schedule will trigger every day.
//
// Schedule respects timezone given in previous trigger point.
func NewDailySchedule(prev time.Time, daydelay uint32) *DailySchedule {
	var s DailySchedule
	s.init(prev, daydelay)
	return &s
}

func (s *DailySchedule) init(prev time.Time, daydelay uint32) {
	h, m, ss := prev.Clock()

	s.prev = MicroTime(prev.UnixMicro())
	s.tz = prev.Location()
	s.local = uint32(60*(60*h+m) + ss)
	s.daydelay = daydelay
}

func (s *DailySchedule) Type() RepTaskType {
	return RepTaskDaily
}

func (s *DailySchedule) Timezone() *time.Location {
	return s.tz
}

func (s *DailySchedule) Next(now time.Time) time.Time {
	nowts := MicroTime(now.UnixMicro())
	delay := 86400 * 1000000 * (uint64(s.daydelay) + 1)

	if s.prev == 0 {
		panic("stub")
	}

	if nowts == 0 || now.IsZero() {
		return s.prev.Time().In(s.tz)
	}

	if nowts == s.prev {
		ts := s.prev
		ts += MicroTime(delay)
		s.prev = ts
		return time.UnixMicro(int64(ts)).In(s.tz)
	}

	if nowts > s.prev {
		ts := s.prev
		for ts < nowts {
			ts += MicroTime(delay)
		}
		s.prev = ts
		return time.UnixMicro(int64(ts)).In(s.tz)
	}

	ts := s.prev
	for {
		if ts-nowts < MicroTime(delay) {
			s.prev = ts
			return time.UnixMicro(int64(ts)).In(s.tz)
		}

		ts -= MicroTime(delay)
	}
}

type DailyScheduleInfo struct {
	// Number of seconds from local day start.
	Local uint32 `json:"local"`

	// Zero is valid value meaning that schedule triggers every day.
	DayDelay uint32 `json:"day_delay,omitzero"`
}

func (s *DailySchedule) Marshal() ([]byte, time.Time, error) {
	data, err := json.Marshal(&DailyScheduleInfo{
		Local:    s.local,
		DayDelay: s.daydelay,
	})
	if err != nil {
		return nil, time.Time{}, err
	}

	return data, s.prev.Time(), nil
}

func (s *DailySchedule) Unmarshal(data []byte, next time.Time) error {
	var info DailyScheduleInfo
	err := json.Unmarshal(data, &info)
	if err != nil {
		return err
	}

	s.init(next, info.DayDelay)
	return nil
}
