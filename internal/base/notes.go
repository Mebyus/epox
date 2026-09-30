package base

import (
	"strconv"
	"time"
)

type TopicID uint64

func (i TopicID) String() string {
	if i == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(i), 10)
}

type NotesTopic struct {
	CreateTime time.Time
	UpdateTime time.Time

	Path        string
	Title       string
	Description string

	ID     TopicID
	UserID UserID

	Offset uint64
}

type MessageID uint64

func (i MessageID) String() string {
	if i == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(i), 10)
}

type NotesMessage struct {
	CreateTime time.Time

	// Topic path. Used when adding new message via external API.
	Path string

	Text string

	ID      MessageID
	UserID  UserID
	TopicID TopicID

	Offset uint64
}

// RequestNotes describes request for messages from notes topic.
type RequestNotes struct {
	// Topic path.
	Path string

	UserID  UserID
	TopicID TopicID

	// Offset maximum for loading messages. All loaded messages
	// will have offset strictly lower than this value.
	//
	// Zero value means latest offset.
	Offset uint64

	// Maximum number of messages to load.
	Limit uint16
}
