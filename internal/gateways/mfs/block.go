package mfs

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"

	"github.com/mebyus/epox/internal/dms"
)

type Storage struct {
	// base directory for storing topics
	base string

	topics map[ /* topic name */ string]*Topic

	mu sync.Mutex
}

type Config struct {
	DataDir string

	MaxBlockSize int
}

func New(config *Config) (*Storage, error) {
	base := config.DataDir
	err := os.MkdirAll(base, 0o750)
	if err != nil {
		return nil, err
	}

	s := &Storage{
		base:   base,
		topics: make(map[string]*Topic),
	}
	err = s.init()
	if err != nil {
		return nil, err
	}

	return s, nil
}

type Block struct {
	// filesystem path where block is stored (or should be stored)
	path string

	// list of block entries, each entry is message data
	ents [][]byte

	// not nil only for incomplete block
	// opened in append mode
	file *os.File

	// number of dirty (unsaved) messages in block
	dirty uint32

	// offset of the first message in block
	offset uint32
}

// Topic stores cached blocks and necessary data for managment of
// topic blocks in filesystem.
type Topic struct {
	// contains list of all block offsets
	idx []uint32

	// filesystem path to topic storage directory
	path string

	// most recent block
	tip *Block

	// previous (complete) block
	prev *Block

	// total number of messages in topic
	num int

	mu sync.RWMutex
}

func formatOffset(offset uint32) string {
	return fmt.Sprintf("%08X", offset)
}

func (s *Storage) AddTopic(ctx context.Context, lg *zap.Logger, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	topic := s.topics[name]
	if topic != nil {
		return dms.ErrTopicAlreadyExists
	}

	path := filepath.Join(s.base, name)
	s.topics[name] = &Topic{path: path}
	return nil
}

func (s *Storage) Add(ctx context.Context, lg *zap.Logger, msg *dms.Message) error {
	topic := s.getTopic(msg.Topic)
	if topic == nil {
		return dms.ErrTopicNotFound
	}

	err := topic.add(msg.Data)
	return err
}

func (s *Storage) init() error {
	return nil
}

func (b *Block) sync() error {
	if b.dirty == 0 {
		return nil
	}

	buf := bufio.NewWriter(b.file)

	var errs []error
	for i := len(b.ents) - int(b.dirty); i < len(b.ents); i += 1 {
		data := b.ents[i]

		var binbuf [2]byte
		binary.LittleEndian.PutUint16(binbuf[:], uint16(len(data)))
		_, err := buf.Write(binbuf[:])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		_, err = buf.Write(data)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err := buf.Flush()
	if err != nil {
		errs = append(errs, err)
	}

	b.dirty = 0
	return errors.Join(errs...)
}

func (b *Block) close() error {
	err := b.file.Close()
	b.file = nil
	return err
}

func (t *Topic) add(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.num == 0 {
		// initialize storage for new topic
		err := os.MkdirAll(t.path, 0o750)
		if err != nil {
			return err
		}

		path := filepath.Join(t.path, "00000000")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o640)
		if err != nil {
			return err
		}
		tip := &Block{
			path: path,
			file: file,
		}
		t.tip = tip
		t.idx = append(t.idx, 0)
	}

	const maxBlockSize = 1 << 8
	tip := t.tip
	if len(tip.ents) >= maxBlockSize {
		err := tip.close()
		if err != nil {

		}
		// create new block and swap tip and prev
		t.prev = tip
	}

	tip.ents = append(tip.ents, data)
	tip.dirty += 1
	t.num += 1

	return nil
}

func (s *Storage) getTopic(name string) *Topic {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.topics[name]
}

func (s *Storage) List(ctx context.Context, lg *zap.Logger, opts *dms.ListOptions) ([]dms.Message, error) {
	return nil, nil
}

// returns nil if topic is successfully found
func (s *Storage) FindTopic(ctx context.Context, lg *zap.Logger, name string) error {
	t := s.getTopic(name)
	if t == nil {
		return dms.ErrTopicNotFound
	}
	return nil
}

func (s *Storage) Sync() error {
	return nil
}
