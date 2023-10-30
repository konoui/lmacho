package fat

import (
	"debug/macho"
	"errors"
	"io"
)

// FatHeader presents a header for a fat 32 bit and fat 64 bit
// see /Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/include/mach-o/fat.h
type FatHeader struct {
	Magic uint32
	NArch uint32
}

// FatArchHeader presents an architecture header for a Macho-0 32 bit and 64 bit
type FatArchHeader struct {
	Cpu    macho.Cpu
	SubCpu SubCpu
	Offset uint64
	Size   uint64
	Align  uint32
}

// FatArch presents an object of fat file
type FatArch struct {
	FatArchHeader
	Hidden bool
	sr     *io.SectionReader
}

func (fa *FatArch) Read(b []byte) (int, error) {
	return fa.sr.Read(b)
}

func (fa *FatArch) ReadAt(b []byte, off int64) (int, error) {
	return fa.sr.ReadAt(b, off)
}

func (fa *FatArch) Seek(offset int64, whence int) (int64, error) {
	return fa.sr.Seek(offset, whence)
}

var ErrThin = errors.New("the file is thin file, not fat")

type FormatError struct {
	Err error
}

func (e *FormatError) Error() string {
	return e.Err.Error()
}

func fatHeaderSize() uint64 {
	// sizeof(FatHeader) = uint32 * 2
	return uint64(4 * 2)
}

func fatArchHeaderSize(magic uint32) uint64 {
	if magic == MagicFat64 {
		// sizeof(Fat64ArchHeader) = uint32 * 4 + uint64 * 2
		return uint64(4*4 + 8*2)
	}
	// sizeof(macho.FatArchHeader) = uint32 * 5
	return uint64(4 * 5)
}
