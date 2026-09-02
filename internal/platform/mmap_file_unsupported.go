//go:build !((darwin || linux || freebsd || netbsd || dragonfly || solaris) && !tinygo) && !windows

package platform

import "os"

const fileMappedCodeSupported = false

func mmapFileCodeSegment(*os.File, int64, int) ([]byte, error) { panic("unsupported") }

func munmapFileCodeSegment([]byte) error { panic("unsupported") }
