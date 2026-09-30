package base

import (
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

	// Can be zero if task has no deadline.
	Deadline time.Time

	Title       string
	Description string

	ID     TaskID
	UserID UserID

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
