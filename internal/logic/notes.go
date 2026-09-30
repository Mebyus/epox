package logic

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/base"
)

// AddNotesTopic adds a topic to user notes.
// Sets topic id upon success.
func (g *Logic) AddNotesTopic(ctx context.Context, lg *zap.Logger, topic *base.NotesTopic) error {
	topic.CreateTime = time.Now()
	return g.db.AddNotesTopic(ctx, lg, topic)
}

func (g *Logic) GetNotesMessages(ctx context.Context, lg *zap.Logger, req *base.RequestNotes) ([]base.NotesMessage, error) {
	id, offset, err := g.db.LookupNotesTopicByPath(ctx, lg, req.UserID, req.Path)
	if err != nil {
		return nil, err
	}
	if offset == 0 {
		return nil, nil
	}

	if req.Limit == 0 || req.Limit > 64 {
		req.Limit = 64
	}
	req.TopicID = id

	return g.db.GetNotesMessages(ctx, lg, req)
}
