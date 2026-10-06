package base

import (
	"cmp"
	"slices"
	"strconv"
	"time"
)

type TaskID uint64

func (i TaskID) String() string {
	if i == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(i), 10)
}

// ActiveTask represents a user task which should be
// shown as active task to user.
type ActiveTask struct {
	Task
}

// HistoryTask represents a user task which is no longer
// active for various possible reasons.
type HistoryTask struct {
	Task

	State TaskState
}

type Task struct {
	Tags []Tag

	CreateTime time.Time
	UpdateTime time.Time

	Title       string
	Description string

	ID     TaskID
	UserID UserID

	// Can be zero if task has no deadline.
	Deadline MicroTime

	// Can be zero if task already started.
	StartTime MicroTime

	// Higher values mean higher importance.
	Importance uint32

	// Current task numerical progress out of MaxProgress.
	Progress uint32

	// Equals zero if task does not have progress indicator.
	MaxProgress uint32
}

type TaskState uint32

// DO NOT change order of constants in this block.
const (
	TaskActive TaskState = iota
	TaskDone
	TaskCanceled
	TaskExpired
)

var taskStateText = [...]string{
	TaskActive:   "active",
	TaskDone:     "done",
	TaskCanceled: "canceled",
	TaskExpired:  "expired",
}

func (s TaskState) String() string {
	return taskStateText[s]
}

type RequestChangeTaskState struct {
	ID     TaskID
	UserID UserID
	State  TaskState
}

// ExpiredTaskChore is used to issue a job for marking
// particular task as expired.
type ExpiredTaskChore struct {
	// Will be saved as update time for expired task.
	Time time.Time

	ID TaskID
}

// SortTasksByUrgency most urgent tasks are placed first.
//
// Urgency is calcutated dynamically based on importance and
// how close deadline is to provided now instant.
func SortTasksByUrgency(tasks []Task, now MicroTime) {
	if len(tasks) < 2 {
		return
	}

	slices.SortFunc(tasks, func(a, b Task) int {
		var as bool // task a started
		if a.StartTime == 0 {
			as = true
		} else {
			as = now >= a.StartTime
		}

		var bs bool // task b started
		if b.StartTime == 0 {
			bs = true
		} else {
			bs = now >= b.StartTime
		}

		if as && !bs {
			return -1
		}
		if !as && bs {
			return 1
		}
		if !as && !bs {
			if a.Importance == b.Importance {
				return b.CreateTime.Compare(a.CreateTime)
			}
			return cmp.Compare(b.Importance, a.Importance)
		}

		// both tasks already started

		if a.Deadline != 0 && b.Deadline == 0 {
			return -1
		}
		if a.Deadline == 0 && b.Deadline != 0 {
			return 1
		}
		if a.Deadline == 0 && b.Deadline == 0 {
			if a.Importance == b.Importance {
				return b.CreateTime.Compare(a.CreateTime)
			}
			return cmp.Compare(b.Importance, a.Importance)
		}

		// both tasks have deadline

		scoreA := calcUrgencyScore(a.Importance, uint64(a.Deadline-now))
		scoreB := calcUrgencyScore(b.Importance, uint64(b.Deadline-now))

		if scoreA == scoreB {
			return b.CreateTime.Compare(a.CreateTime)
		}

		return cmp.Compare(scoreB, scoreA)
	})
}

// higher score means higher urgency
//
// timeleft provided in microseconds
func calcUrgencyScore(importance uint32, timeleft uint64) uint32 {
	hours := timeleft / (1000000 * 60 * 60)

	var s uint32
	switch {
	case hours < 2:
		s = 10
	case hours < 12:
		s = 5
	case hours < 48:
		s = 2
	case hours < 7*24:
		s = 1
	}

	return importance + s
}
