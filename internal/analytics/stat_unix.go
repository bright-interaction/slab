//go:build unix

package analytics

import (
	"os"
	"syscall"
)

// statInode returns the inode number of an open file on Unix systems.
// Used to detect log rotation: when nginx truncates+reopens, the inode
// changes and we know to reopen our handle.
func statInode(f *os.File) (uint64, error) {
	st, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return statInodeFromInfo(st), nil
}

func statInodeFromInfo(fi os.FileInfo) uint64 {
	if sys, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(sys.Ino)
	}
	return 0
}
