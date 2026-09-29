package base

import "strconv"

// TagID
//
// Zero value means empty/no id.
type TagID uint64

func (i TagID) String() string {
	if i == 0 {
		return ""
	}
	return strconv.FormatUint(uint64(i), 10)
}

type Tag struct {
	Name string
	ID   TagID
}
