//go:build (darwin || linux || freebsd || netbsd || dragonfly || solaris) && !tinygo

package platform

import (
	"os"

	"golang.org/x/sys/unix"
)

const fileMappedCodeSupported = true

func mmapFileCodeSegment(f *os.File, off int64, size int) ([]byte, error) {
	return unix.Mmap(int(f.Fd()), off, size, unix.PROT_READ|unix.PROT_EXEC, unix.MAP_PRIVATE)
}

func munmapFileCodeSegment(b []byte) error {
	return unix.Munmap(b)
}
