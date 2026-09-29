package froll

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const debug = true

// Worker handles incoming data from Roller and writes it
// to underlying active file
type worker struct {
	// active file name (without directory path)
	name string

	// active file prefix (for path) without generated timestamp,
	// extension, number, etc.
	prefix string

	// directory path where rotating files are stored
	dir string

	// rotated file extension (without compression)
	//
	// example: ".log"
	ext string

	// compressed file suffix
	//
	// example: ".log.gz"
	csuf string

	// max allowed size in bytes
	maxsize uint64

	// current size of active file in bytes
	size uint64

	// active file used for writing
	file *os.File

	// for incoming data-to-be-written
	tap <-chan *buffer

	// for sync signal
	ss <-chan struct{}

	// for close signal
	cs <-chan struct{}

	// send swap entries to gc
	swap chan swap

	// Access to reusable byte buffers.
	//
	// Shared with Roller.
	pool *sync.Pool

	// max number (excluding active one) of old files to keep
	maxnum uint32
}

// entry represents rotated file
type entry struct {
	modtime time.Time

	// local file name (without directory path)
	name string

	size uint64

	compressed bool
}

func sortEntries(entries []entry) {
	if len(entries) <= 1 {
		return
	}

	slices.SortFunc(entries, func(a, b entry) int {
		return a.modtime.Compare(b.modtime)
	})
}

// returns prepared slots for gc
func (w *worker) init() ([]string, error) {
	if w.dir != "" {
		err := os.MkdirAll(w.dir, 0o755)
		if err != nil {
			return nil, err
		}
	}

	list, err := os.ReadDir(w.dir)
	if err != nil {
		return nil, err
	}

	entries := make([]entry, 0, w.maxnum)
	for _, e := range list {
		if !e.Type().IsRegular() {
			continue
		}

		name := e.Name()
		var skip bool
		var compressed bool
		if strings.HasSuffix(name, w.ext) {
			// do not skip
		} else if strings.HasSuffix(name, w.csuf) {
			// do not skip
			compressed = true
		} else {
			skip = true
		}

		if skip {
			continue
		}

		info, err := e.Info()
		if err != nil {
			return nil, err
		}

		size := uint64(info.Size())
		if size == 0 {
			err := os.Remove(filepath.Join(w.dir, name))
			if err != nil {
				if debug {
					fmt.Printf("[error] remove bad entry: %v\n", err)
				}
			}
			continue
		}

		entries = append(entries, entry{
			name:       name,
			modtime:    info.ModTime(),
			size:       size,
			compressed: compressed,
		})
	}
	if len(entries) == 0 {
		w.rotate()
		slots := make([]string, w.maxnum)
		return slots, nil
	}

	sortEntries(entries)
	last := entries[len(entries)-1]
	if last.compressed {
		// create new active file
		// TODO: return error on fail
		w.rotate()
	} else if last.size >= w.maxsize {
		// TODO: send swap with old file somehow
		w.rotate()
	} else {
		path := filepath.Join(w.dir, last.name)
		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			if debug {
				fmt.Printf("[error] open existing file: %v\n", err)
			}
			w.rotate()
		} else {
			entries = entries[:len(entries)-1]
			w.file = file
			w.name = last.name
		}
		if len(entries) == 0 {
			slots := make([]string, w.maxnum)
			return slots, nil
		}
	}

	slots := make([]string, w.maxnum)
	if uint32(len(entries)) <= w.maxnum {
		k := w.maxnum - uint32(len(entries))
		for i := uint32(0); i < uint32(len(entries)); i += 1 {
			slots[k+i] = filepath.Join(w.dir, entries[i].name)
		}
		return slots, nil
	}

	// remove oldest entries until only maxnum remains
	k := uint32(len(entries)) - w.maxnum // number of entries that needs to be removed
	for i := uint32(0); i < w.maxnum; i += 1 {
		slots[i] = filepath.Join(w.dir, entries[k+i].name)
	}
	for i := range k {
		path := filepath.Join(w.dir, entries[i].name)
		err := os.Remove(path)
		if err != nil {
			if debug {
				fmt.Printf("[error] remove old file: %v\n", err)
			}
		} else {
			if debug {
				fmt.Printf("[debug] removed old file \"%s\"\n", path)
			}
		}
	}

	return slots, nil
}

func (w *worker) watch() {
	// fsync ticker for periodic system cache flush
	st := time.NewTicker(250 * time.Millisecond)

	for {
		select {
		case buf := <-w.tap:
			if debug {
				// fmt.Printf("[debug] worker received %d bytes of data\n", len(data))
			}
			w.write(buf.bytes())
			w.putBuffer(buf)
		case <-w.ss:
			w.sync()
		case <-w.cs:
			st.Stop()
			w.close()
		case <-st.C:
			w.sync()
		}
	}
}

func (w *worker) write(data []byte) {
	if w.file == nil {
		// not initialized properly or explicitly closed
		return
	}

	if w.size+uint64(len(data)) > w.maxsize {
		w.rotate()
	}

	n, err := w.file.Write(data)
	w.size += uint64(n)

	if err != nil {
		if debug {
			fmt.Printf("[error] write data: %v\n", err)
		}
		// TODO: do something
	}
}

// create a new file and switch active pointer to it
// if successfull
func (w *worker) rotate() {
	path, name := w.getNewPath()
	file, err := os.Create(path)
	if err != nil {
		if debug {
			fmt.Printf("[error] create new active file: %v\n", err)
		}
		return
	}

	old := w.file
	oldName := w.name
	w.name = name
	w.file = file
	w.size = 0

	var s swap
	if old == nil {
		s = swap{new: name}
	} else {
		s = swap{
			new:  name,
			old:  oldName,
			file: old,
		}
	}
	w.swap <- s
}

func (w *worker) sync() {
	err := w.file.Sync()
	if err != nil {
		if debug {
			fmt.Printf("[error] sync active file: %v\n", err)
		}
		// TODO: do something
	}
}

func (w *worker) close() {
	if w.file == nil {
		return
	}

	w.file.Close()
	w.file = nil
}

// generate path for a new active file
//
// returns path and name separately
func (w *worker) getNewPath() (string, string) {
	const format = "2006-01-02T15-04-05.999"

	var g strings.Builder
	g.Grow(len(w.prefix) + 1 + len(format) + len(w.ext))
	g.WriteString(w.prefix)
	g.WriteByte('-')
	g.WriteString(time.Now().Format(format))
	g.WriteString(w.ext)

	path := g.String()
	name := path[len(w.dir)+1:] // trim leading dir path + "/" to get file name

	return path, name
}
