package changelog

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/qiniu/encoding/tab"
	// "github.com/qiniu/log"
)

/* -----------------------------------------------

<url>\n

// ---------------------------------------------*/

type Logger struct {
	File io.Writer
}

func OpenNew(w io.Writer) (p Logger) {

	return Logger{w}
}

func Open(fname string) (p Logger, urls []string, err error) {

	urls, err = loadEntriesAndBackup(fname)
	if err != nil {
		return
	}

	f, err := os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	p = Logger{f}
	return
}

func (p Logger) Close() error {

	if f, ok := p.File.(io.Closer); ok {
		f.Close()
	}
	return nil
}

func (p Logger) Put(s string) {

	encodedStr := tab.Escape(s)
	fmt.Fprintln(p.File, encodedStr)
}

// -----------------------------------------------

func loadEntriesAndBackup(fname string) (urls []string, err error) {

	fnamebak := fname + ".bak"
	urls, err = LoadUrls(fname)
	if err != nil {
		urls, err = LoadUrls(fnamebak)
		if err != nil {
			if err == syscall.ENOENT {
				err = nil
			} else {
				return
			}
		}
	} else {
		os.Remove(fnamebak)
		os.Rename(fname, fnamebak)
	}

	if len(urls) == 0 {
		return
	}

	fnamenew, err := saveEntries(fname, urls)
	if err != nil {
		return
	}

	err = os.Rename(fnamenew, fname)
	return
}

func saveEntries(fname string, urls []string) (fnamenew string, err error) {

	fnamenew = fname + ".new"
	f, err := os.OpenFile(fnamenew, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	w := bufio.NewWriterSize(f, 4096*2)
	p2 := Logger{w}
	for _, e := range urls {
		p2.Put(e)
	}
	err = w.Flush()
	return
}

func LoadUrls(fname string) (urls []string, err error) {

	f, err := os.Open(fname)
	if err != nil {
		if e, ok := err.(*os.PathError); ok && e.Err == syscall.ENOENT {
			return nil, syscall.ENOENT
		}
		return
	}
	defer f.Close()

	br := bufio.NewReaderSize(f, 4096)
	for {
		line, isPrefix, err2 := br.ReadLine()
		err = err2
		if err != nil || isPrefix {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
		line1 := string(line) + "\n"
		var url string
		_, err = fmt.Sscanln(line1, &url)
		if err != nil {
			return
		}
		urlUnescape, err2 := tab.Unescape(url)
		err = err2
		if err != nil {
			return
		}
		urls = append(urls, urlUnescape)
	}

	return
}

// -----------------------------------------------
