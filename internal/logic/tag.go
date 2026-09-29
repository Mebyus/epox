package logic

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/base"
)

// SaveTags saves tags into persistent storage and returns them back
// with assigned ids. This operation can be done on already existing
// tags in which case it returns stored ids for such tags.
func (g *Logic) SaveTags(ctx context.Context, lg *zap.Logger, list []string) ([]base.Tag, error) {
	return g.tags.save(ctx, lg, list)
}

type TagStorage interface {
	GetTags(ctx context.Context, lg *zap.Logger) ([]base.Tag, error)
	SaveTags(ctx context.Context, lg *zap.Logger, list []string) ([]base.Tag, error)
}

type TagCache struct {
	mu sync.RWMutex

	// stores all tags loaded in cache
	tags []base.Tag

	s TagStorage

	m map[ /* tag name */ string]base.TagID

	// maps tag id to its name
	d map[base.TagID]string
}

func (c *TagCache) init(ctx context.Context, lg *zap.Logger, s TagStorage) error {
	tags, err := s.GetTags(ctx, lg)
	if err != nil {
		return err
	}

	m := make(map[string]base.TagID, len(tags))
	d := make(map[base.TagID]string, len(tags))
	for _, t := range tags {
		m[t.Name] = t.ID
		d[t.ID] = t.Name
	}

	c.tags = tags
	c.m = m
	c.d = d
	c.s = s

	lg.Debug("loaded tags into cache", zap.Int("num", len(tags)))
	return nil
}

func (c *TagCache) save(ctx context.Context, lg *zap.Logger, list []string) ([]base.Tag, error) {
	if len(list) == 0 {
		return nil, nil
	}

	tags, list := c.separate(list)
	if len(list) == 0 {
		return tags, nil
	}

	stored, err := c.s.SaveTags(ctx, lg, list)
	if err != nil {
		return nil, err
	}
	c.append(stored)

	tags = append(tags, stored...)
	return tags, nil
}

// Fills tag names based on their ids in the given slice.
//
// Slice if modified in place.
func (c *TagCache) fill(tags []base.Tag) {
	if len(tags) == 0 {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range len(tags) {
		t := &tags[i]
		if t.Name == "" {
			t.Name = c.d[t.ID]
		}
	}
}

// Fills tag ids based on their names.
//
// Returns an error if there is a tag with unknown name.
func (c *TagCache) valid(tags []base.Tag) error {
	if len(tags) == 0 {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	for i := range len(tags) {
		t := &tags[i]
		if t.ID == 0 {
			id, ok := c.m[t.Name]
			if !ok {
				return base.ErrTagNotFound
			}
			t.ID = id
		}
	}

	return nil
}

// separate checks each tag in the given list and separates them
// into two lists. First list contains tags already stored in cache
// and second list contains tags not stored in cache.
func (c *TagCache) separate(list []string) ([]base.Tag, []string) {
	if len(list) == 0 {
		return nil, nil
	}

	var unknown []string
	tags := make([]base.Tag, 0, len(list))

	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, name := range list {
		id, ok := c.m[name]
		if ok {
			tags = append(tags, base.Tag{
				Name: name,
				ID:   id,
			})
		} else {
			unknown = append(unknown, name)
		}
	}

	return tags, unknown
}

// append stores new tags in cache.
func (c *TagCache) append(tags []base.Tag) {
	if len(tags) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, t := range tags {
		_, ok := c.m[t.Name]
		if !ok {
			c.m[t.Name] = t.ID
			c.d[t.ID] = t.Name
			c.tags = append(c.tags, t)
		}
	}
}
