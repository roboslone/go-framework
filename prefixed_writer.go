package framework

import "io"

type PrefixedWriter struct {
	w      io.Writer
	prefix []byte
}

func NewPrefixedWriter(w io.Writer, prefix string) *PrefixedWriter {
	return &PrefixedWriter{w: w, prefix: []byte(prefix)}
}

func (w *PrefixedWriter) Write(p []byte) (int, error) {
	msg := make([]byte, len(p)+len(w.prefix))
	msg = append(msg, w.prefix...)
	msg = append(msg, p...)

	if _, err := w.w.Write(msg); err != nil {
		return 0, err
	}
	return len(p), nil
}
