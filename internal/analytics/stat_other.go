//go:build !unix

package analytics

import "os"

// Non-Unix fallback: detect rotation via size-shrink only (we return 0 and
// the parser's other rotation check still works).
func statInode(f *os.File) (uint64, error) {
	_, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return 0, nil
}

func statInodeFromInfo(fi os.FileInfo) uint64 { return 0 }
