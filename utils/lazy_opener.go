package utils

import (
	"io"
	"os"
)

type lazyOpener struct {
	name      string
	overwrite bool
	f         *os.File
	err       error
}

func NewLazyOpener(name string, overwrite bool) io.WriteCloser {
	return &lazyOpener{name: name, overwrite: overwrite}
}

func (l *lazyOpener) Write(p []byte) (n int, err error) {
	if l.f == nil && l.err == nil {
		oFlags := os.O_WRONLY | os.O_CREATE
		perms := os.FileMode(0o660)

		if l.overwrite {
			stat, err := os.Stat(l.name)
			if err != nil {
				return 0, err
			}
			perms = stat.Mode()
			oFlags = oFlags | os.O_TRUNC
		} else {
			oFlags = oFlags | os.O_EXCL
		}

		l.f, l.err = os.OpenFile(l.name, oFlags, perms)
	}
	if l.err != nil {
		return 0, l.err
	}
	return l.f.Write(p)
}

func (l *lazyOpener) Close() error {
	if l.f != nil {
		return l.f.Close()
	}
	return nil
}
