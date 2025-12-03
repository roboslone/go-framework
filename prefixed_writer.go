package framework

import (
	"bytes"
	"io"
)

var (
	nl = []byte("\n")
)

type PrefixedWriter struct {
	w      io.Writer
	prefix []byte
}

func NewPrefixedWriter(w io.Writer, prefix string) *PrefixedWriter {
	return &PrefixedWriter{w: w, prefix: []byte(prefix)}
}

func (w *PrefixedWriter) Write(p []byte) (int, error) {
	for line := range bytes.SplitSeq(p, nl) {
		msg := make([]byte, len(line)+len(w.prefix)+len(nl))
		msg = append(msg, w.prefix...)
		msg = append(msg, line...)
		msg = append(msg, nl...)
		if _, err := w.w.Write(msg); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}
