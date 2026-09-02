// Package platform includes runtime-specific code needed for the compiler or otherwise.
package platform

import (
	"os"
	"runtime"
	"sync"
	"unsafe"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
)

// CompilerSupported includes constraints here and also the assembler.
func CompilerSupported() bool {
	return CompilerSupports(api.CoreFeaturesV2)
}

func CompilerSupports(features api.CoreFeatures) bool {
	if !compilerPlatformSupports(features) {
		return false
	}
	// Won't panic
	return executableMmapSupported()
}

func compilerPlatformSupports(features api.CoreFeatures) bool {
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "netbsd", "windows":
		if runtime.GOARCH == "arm64" {
			if features.IsEnabled(experimental.CoreFeaturesThreads) {
				return CpuFeatures.Has(CpuFeatureArm64Atomic)
			}
			return true
		}
		fallthrough
	case "dragonfly", "solaris", "illumos":
		return runtime.GOARCH == "amd64" && CpuFeatures.Has(CpuFeatureAmd64SSE4_1)
	default:
		return false
	}
}

// MmapCodeSegment allocates and returns a byte slice to copy executable code into.
//
// See https://man7.org/linux/man-pages/man2/mmap.2.html for mmap API and flags.
func MmapCodeSegment(size int) ([]byte, error) {
	if size == 0 {
		panic("BUG: MmapCodeSegment with zero length")
	}
	return mmapCodeSegment(size)
}

// MunmapCodeSegment unmaps the given memory region.
func MunmapCodeSegment(code []byte) error {
	if len(code) == 0 {
		panic("BUG: MunmapCodeSegment with zero length")
	}
	if unmapFileCodeSegment(code) {
		return nil
	}
	return munmapCodeSegment(code)
}

// fileMapped records the executable segments mapped from files, so that
// MunmapCodeSegment releases them the right way.
var (
	fileMappedMu sync.Mutex
	fileMapped   = map[uintptr]int{}
)

func rememberFileMapped(b []byte) {
	fileMappedMu.Lock()
	fileMapped[uintptr(unsafe.Pointer(&b[0]))] = len(b)
	fileMappedMu.Unlock()
}

func unmapFileCodeSegment(code []byte) bool {
	addr := uintptr(unsafe.Pointer(&code[0]))
	fileMappedMu.Lock()
	n, ok := fileMapped[addr]
	if ok {
		delete(fileMapped, addr)
	}
	fileMappedMu.Unlock()
	if !ok {
		return false
	}
	if err := munmapFileCodeSegment(code[:n]); err != nil {
		panic(err)
	}
	return true
}

// FileMappedCodeSupported reports whether MmapFileCodeSegment works here.
// WAZERO_NO_FILE_MAPPED_CODE=1 forces the copying path (A/B measurements).
func FileMappedCodeSupported() bool {
	return fileMappedCodeSupported && os.Getenv("WAZERO_NO_FILE_MAPPED_CODE") == ""
}

// MmapFileCodeSegment maps size bytes of f starting at off (a multiple of
// 64 KiB) as read-only executable memory, privately. The pages are demand
// loaded and shared with other processes mapping the same file.
func MmapFileCodeSegment(f *os.File, off int64, size int) ([]byte, error) {
	if size == 0 {
		panic("BUG: MmapFileCodeSegment with zero length")
	}
	b, err := mmapFileCodeSegment(f, off, size)
	if err != nil {
		return nil, err
	}
	rememberFileMapped(b)
	return b, nil
}

func executableMmapSupported() bool {
	seg, err := MmapCodeSegment(1)
	if err != nil {
		return false
	}
	defer func() {
		_ = MunmapCodeSegment(seg)
	}()
	if err := MprotectCodeSegment(seg); err != nil {
		return false
	}
	return true
}
