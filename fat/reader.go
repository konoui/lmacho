package fat

import (
	"debug/macho"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/konoui/lmacho/cpu"
)

type Reader struct {
	r                 *io.SectionReader
	FatHeader         FatHeader
	firstObjectOffset uint64
	nextNArch         uint32
}

func NewReader(r io.ReaderAt) (*Reader, error) {
	sr := io.NewSectionReader(r, 0, 1<<63-1)

	var ff FatHeader
	err := binary.Read(sr, binary.BigEndian, &ff.Magic)
	if err != nil {
		return nil, &FormatError{errors.New("error reading magic number")}
	}

	if ff.Magic != macho.MagicFat && ff.Magic != MagicFat64 {
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], ff.Magic)
		leMagic := binary.LittleEndian.Uint32(buf[:])
		if leMagic == macho.Magic32 || leMagic == macho.Magic64 {
			return nil, ErrThin
		}
		return nil, &FormatError{errors.New("invalid magic number")}
	}

	err = binary.Read(sr, binary.BigEndian, &ff.NArch)
	if err != nil {
		return nil, &FormatError{errors.New("invalid fat_header")}
	}

	if ff.NArch < 1 {
		return nil, &FormatError{errors.New("file contains no images")}
	}

	return &Reader{r: sr, FatHeader: ff, nextNArch: 1}, nil
}

func (r *Reader) Next() (*FatArch, error) {
	defer func() {
		r.nextNArch++
	}()
	magic := r.FatHeader.Magic
	if r.nextNArch <= r.FatHeader.NArch {
		hdr, err := readFatArchHeader(r.r, magic)
		if err != nil {
			return nil, &FormatError{err}
		}

		fa := &FatArch{
			sr:            io.NewSectionReader(r.r, int64(hdr.offset), int64(hdr.Size)),
			FatArchHeader: *hdr,
			Hidden:        false,
		}

		if r.firstObjectOffset == 0 {
			r.firstObjectOffset = hdr.offset
		}

		return fa, nil
	}

	// hidden arches
	nextObjectOffset := fatHeaderSize() + fatArchHeaderSize(magic)*uint64(r.nextNArch-1)
	// require to add fatArchHeaderSize, to read the header
	if nextObjectOffset+fatArchHeaderSize(magic) > r.firstObjectOffset {
		return nil, io.EOF
	}

	hr := io.NewSectionReader(r.r, int64(nextObjectOffset), int64(fatArchHeaderSize(magic)))
	fatHdr, err := readFatArchHeader(hr, magic)
	if err != nil {
		return nil, &FormatError{fmt.Errorf("hideARM64: %w", err)}
	}

	if fatHdr.Cpu != cpu.TypeArm64 {
		// TODO handle error
		return nil, io.EOF
	}

	return &FatArch{
		sr:            io.NewSectionReader(r.r, int64(fatHdr.offset), int64(fatHdr.Size)),
		FatArchHeader: *fatHdr,
		Hidden:        true,
	}, nil
}

func readFatArchHeader(r io.Reader, magic uint32) (*FatArchHeader, error) {
	if magic == MagicFat64 {
		var fatHdr fatArch64Header
		err := binary.Read(r, binary.BigEndian, &fatHdr)
		if err != nil {
			return nil, errors.New("invalid fat arch64 header")
		}

		return &FatArchHeader{
			Cpu:    fatHdr.Cpu,
			SubCpu: fatHdr.SubCpu,
			Align:  fatHdr.Align,
			Size:   fatHdr.Size,
			offset: fatHdr.offset,
		}, nil
	}

	var fatHdr macho.FatArchHeader
	err := binary.Read(r, binary.BigEndian, &fatHdr)
	if err != nil {
		return nil, errors.New("invalid fat arch header")
	}
	return &FatArchHeader{
		Cpu:    fatHdr.Cpu,
		SubCpu: fatHdr.SubCpu,
		Align:  fatHdr.Align,
		Size:   uint64(fatHdr.Size),
		offset: uint64(fatHdr.Offset),
	}, nil
}
