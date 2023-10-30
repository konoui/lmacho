package fat

import (
	"bufio"
	"io"
)

type Writer struct {
	w io.Writer
}

func NewWriter(w io.Writer, f FatHeader) *Writer {
	return &Writer{w: bufio.NewWriter(w)}
}

func (w *Writer) WriteHeader(f FatArchHeader) error {
	return nil
}

func (w *Writer) Write(p []byte) (int, error) {
	return w.w.Write(p)
}
