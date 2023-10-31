package fat

import (
	"debug/macho"
	"errors"
	"io"

	"github.com/konoui/lmacho/ar"
	"github.com/konoui/lmacho/cpu"
)

const MagicFat = macho.MagicFat

// FatHeader presents a header for a fat 32 bit and fat 64 bit
// see /Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/include/mach-o/fat.h
type FatHeader struct {
	Magic uint32
	NArch uint32
}

// FatArchHeader presents an architecture header for a Macho-0 32 bit and 64 bit
type FatArchHeader struct {
	Cpu    cpu.Cpu
	SubCpu cpu.SubCpu
	offset uint64 // internal properties to calculate in this package
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

type FatFile struct {
	FatHeader
	Arches []FatArch
}

// NewFatFile is wrapper for Fat Reader
func NewFatFile(ra io.ReaderAt) (*FatFile, error) {
	r, err := NewReader(ra)
	if err != nil {
		return nil, err
	}

	fa := &FatFile{
		Arches:    make([]FatArch, 0),
		FatHeader: r.FatHeader,
	}

	for {
		a, err := r.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		fa.Arches = append(fa.Arches, *a)
	}
	return fa, nil
}

func NewFatArch(sr *io.SectionReader) (*FatArch, error) {
	getFromAr := func(ra io.ReaderAt) (*macho.File, error) {
		r, err := ar.NewReader(sr)
		if err != nil {
			return nil, err
		}

		for {
			f, err := r.Next()
			if err != nil {
				return nil, err
			}

			if f.Name == ar.PrefixSymdef {
				continue
			}

			return macho.NewFile(sr)
		}
	}

	var hdr *FatArchHeader
	errs := make([]error, 0, 2)
	for _, getter := range []func(io.ReaderAt) (*macho.File, error){
		macho.NewFile,
		getFromAr,
	} {
		m, err := getter(sr)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		hdr = &FatArchHeader{
			Cpu:    m.Cpu,
			SubCpu: m.SubCpu,
			Align:  SegmentAlignBit(m),
			Size:   uint64(sr.Size()),
		}

		if _, err := sr.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		break
	}

	if hdr == nil {
		return nil, errors.Join(errs...)
	}

	arch := FatArch{
		FatArchHeader: *hdr,
		sr:            sr,
	}

	return &arch, nil
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
