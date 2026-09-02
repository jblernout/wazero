package platform

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Off on Windows: a file with a mapped section cannot be deleted or
// replaced until the last view is unmapped, which happens in a finalizer,
// so a cache directory could not be cleaned or refreshed while an engine
// runs. The copying path is used instead; Linux gets the shared mapping.
const fileMappedCodeSupported = false

func mmapFileCodeSegment(f *os.File, off int64, size int) ([]byte, error) {
	// An executable mapping needs a handle opened with execute access, which
	// os.Open does not request: reopen the file by name.
	name, err := windows.UTF16PtrFromString(f.Name())
	if err != nil {
		return nil, err
	}
	fh, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_EXECUTE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(fh)
	maxSize := uint64(off) + uint64(size)
	h, err := windows.CreateFileMapping(fh, nil, windows.PAGE_EXECUTE_READ, uint32(maxSize>>32), uint32(maxSize), nil)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(h) // the view keeps the mapping alive
	p, err := windows.MapViewOfFile(h, windows.FILE_MAP_READ|windows.FILE_MAP_EXECUTE, uint32(uint64(off)>>32), uint32(uint64(off)), uintptr(size))
	if err != nil {
		return nil, err
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(p)), size), nil
}

func munmapFileCodeSegment(b []byte) error {
	return windows.UnmapViewOfFile(uintptr(unsafe.Pointer(&b[0])))
}
