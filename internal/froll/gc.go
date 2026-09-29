package froll

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// gc manages inactive rotations of file:
//
// List of gc responsibilities:
//   - close open files upon swap
//   - compress new entries
//   - delete oldest entries if number limit is reached
//   - update symlink to active file
type gc struct {
	// contains paths to old files which are waiting to be
	// deleted after their storage limit is reached
	cbuf cycle

	// Directory path where rotating files are stored.
	dir string

	// Path to symlink file.
	symlink string

	// Receive new (uncompressed) entries.
	tap <-chan swap
}

type swap struct {
	// Newly created file. Should attach symlink to it.
	//
	// Always not empty.
	//
	// File base name (without directory path).
	new string

	// File that was rotated out, when new file was created.
	// Should be uncompressed file.
	//
	// May be empty if there was no old file before rotation.
	//
	// File base name (without directory path).
	old string

	// Open file that was rotated out. Should be closed
	// in background job.
	file *os.File
}

func (g *gc) watch() {
	// sync ticker
	st := time.NewTicker(531 * time.Millisecond)

	for {
		select {
		case s := <-g.tap:
			g.swap(s)
		case <-st.C:
			g.sync()
		}
	}
}

func (g *gc) swap(s swap) {
	err := swapSymlink(g.symlink, s.new)
	if err != nil {
		if debug {
			fmt.Printf("[error] swap symlink: %v\n", err)
		}
	}

	if s.file != nil {
		err = s.file.Close()
		if err != nil {
			if debug {
				fmt.Printf("[error] close rotated file: %v\n", err)
			}
		}
	}

	if s.old != "" {
		path := g.compress(s.old)
		if path != "" {
			g.remove(g.cbuf.push(path))
		}
	}
}

func swapSymlink(symlink, target string) error {
	tmp := symlink + ".tmp"

	err := os.Symlink(target, tmp)
	if err != nil {
		return err
	}

	err = os.Rename(tmp, symlink)
	if err != nil {
		cerr := os.Remove(tmp)
		if cerr != nil {
			if debug {
				fmt.Printf("[error] remove temp symlink: %v\n", err)
			}
		}
		return err
	}

	return nil
}

// Remove old file from rotation.
func (g *gc) remove(path string) {
	err := os.Remove(path)
	if err != nil {
		if debug {
			fmt.Printf("[error] remove rotated file: %v\n", err)
		}
	}
}

// Compress file inside rotation directory.
// This operation creates a new compressed version of
// specified file and deletes it afterward.
//
// Can accept empty file name in which case nothing
// will be done.
//
// Accepts base file name (without directory path).
//
// Returns path to compressed file. May return empty
// string if there was an error or called with empty
// file name.
func (g *gc) compress(name string) string {
	if name == "" {
		return ""
	}

	path := filepath.Join(g.dir, name)
	cpath := filepath.Join(g.dir, getCompressedName(name))

	src, err := os.Open(path)
	if err != nil {
		if debug {
			fmt.Printf("[error] open rotated file: %v\n", err)
		}
		return ""
	}
	defer src.Close()

	dst, err := os.Create(cpath)
	if err != nil {
		if debug {
			fmt.Printf("[error] create compressed file: %v\n", err)
		}
		return ""
	}
	defer dst.Close()

	w := gzip.NewWriter(dst)
	defer w.Close()

	_, err = io.Copy(w, src)
	if err != nil {
		if debug {
			fmt.Printf("[error] write compressed data: %v\n", err)
		}
		// TODO: remove malformed compressed file?
		return ""
	}

	err = os.Remove(path)
	if err != nil {
		if debug {
			fmt.Printf("[errpr] remove uncompressed file: %v\n", err)
		}
	}

	return cpath
}

func (g *gc) sync() {

}

// Cycle buffer for watched old files.
//
// Examples how slots in buffer work:
// ("^" marks current pos index)
//
//		1 2 3 4 _ _ _  | push 5
//		        ^
//
//		1 2 3 4 5 _ _
//	           ^
//
//
//		1 2 3 4 5 6 7  | push 8
//		^
//
//		8 2 3 4 5 6 7
//		  ^
type cycle struct {
	slots []string

	// index into slots where next push will be placed
	pos uint32
}

func (c *cycle) reset(slots []string) {
	if len(slots) == 0 {
		panic("no slots")
	}

	c.slots = slots
	c.pos = 0
}

// push new string into buffer and return displaced old value
// from the occupied slot
//
// Returned value can be empty string, which usually means that
// slot was empty before the push.
func (c *cycle) push(s string) string {
	old := c.slots[c.pos]
	c.slots[c.pos] = s
	c.pos += 1
	if c.pos >= uint32(len(c.slots)) {
		c.pos = 0
	}
	return old
}

func getCompressedName(name string) string {
	return name + ".gz"
}
