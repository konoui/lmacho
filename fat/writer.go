package fat

import (
	"debug/macho"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/konoui/lmacho/cpu"
)

func Create(w io.Writer, fatArches []FatArch, magic uint32, hideARM64 bool) error {
	if len(fatArches) == 0 {
		return errors.New("file contains no images")
	}

	hdr := makeFatHeader(fatArches, magic, hideARM64)
	if err := sortArches(fatArches, hdr.Magic); err != nil {
		return err
	}

	if err := writeHeaders(w, hdr, fatArches); err != nil {
		return err
	}

	if err := writeArches(w, fatArches, hdr.Magic); err != nil {
		return err
	}

	return nil
}

func writeHeaders(w io.Writer, hdr FatHeader, arches []FatArch) error {
	if err := hasDuplicatesErr(arches); err != nil {
		return err
	}

	if err := checkMaxAlignBit(arches); err != nil {
		return err
	}

	// write a fat header
	// see https://cs.opensource.google/go/go/+/refs/tags/go1.18:src/debug/macho/fat.go;l=45
	if err := binary.Write(w, binary.BigEndian, hdr); err != nil {
		return fmt.Errorf("error write fat_header: %w", err)
	}

	// write fat arch headers
	for _, arch := range arches {
		if err := writeFatArchHeader(w, arch.FatArchHeader, hdr.Magic); err != nil {
			return err
		}
	}
	return nil
}

func writeArches(w io.Writer, arches []FatArch, magic uint32) error {
	firstObjectOffset := fatHeaderSize() + fatArchHeaderSize(magic)*uint64(len(arches))
	offset := firstObjectOffset
	for _, fatArch := range arches {
		if offset < fatArch.offset {
			// write empty data for alignment
			empty := make([]byte, fatArch.offset-offset)
			if _, err := w.Write(empty); err != nil {
				return fmt.Errorf("error alignment: %w", err)
			}
			offset = fatArch.offset
		}

		// write binary data
		if _, err := io.CopyN(w, &fatArch, int64(fatArch.Size)); err != nil {
			return fmt.Errorf("error write binary data: %w", err)
		}
		offset += fatArch.Size
	}

	return nil
}

func writeFatArchHeader(out io.Writer, hdr FatArchHeader, magic uint32) error {
	if magic == MagicFat64 {
		fatArchHdr := fatArch64Header{FatArchHeader: hdr, Reserved: 0}
		if err := binary.Write(out, binary.BigEndian, fatArchHdr); err != nil {
			return fmt.Errorf("error write fat_arch64 header: %w", err)
		}
		return nil
	}

	fatArchHdr := macho.FatArchHeader{
		Cpu:    hdr.Cpu,
		SubCpu: hdr.SubCpu,
		Offset: uint32(hdr.offset),
		Size:   uint32(hdr.Size),
		Align:  hdr.Align,
	}
	if err := binary.Write(out, binary.BigEndian, fatArchHdr); err != nil {
		return fmt.Errorf("error write fat_arch header: %w", err)
	}
	return nil
}

func makeFatHeader(farches []FatArch, magic uint32, hideARM64 bool) FatHeader {
	var found bool
	for _, fatArch := range farches {
		if fatArch.Cpu == cpu.TypeArm {
			found = true
			break
		}
	}

	if !found {
		return FatHeader{
			Magic: magic,
			NArch: uint32(len(farches)),
		}
	}

	narch := uint32(0)
	for i := range farches {
		if farches[i].Cpu == cpu.TypeArm64 {
			continue
		} else {
			narch++
		}
	}

	return FatHeader{
		Magic: magic,
		NArch: narch,
	}
}
