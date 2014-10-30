package namediff

import (
	// "github.com/qiniu/log"
	"sort"
	"strings"
)

func Diff(urls, base []string) (urlsNeedSync []string) {
	sort.Strings(urls)
	sort.Strings(base)

	f := 0
	t := 0
	for f != len(urls) && t != len(base) {
		uf := strings.Trim(urls[f], " ")
		ut := strings.Trim(base[t], " ")
		if uf < ut {
			urlsNeedSync = append(urlsNeedSync, uf)
			f++
		} else if ut < uf {
			t++
		} else {
			f++
			t++
		}
	}

	if f != len(urls) {
		urlsNeedSync = append(urlsNeedSync, urls[f:]...)
	}

	if t != len(base) {
		urlsNeedSync = append(urlsNeedSync, base[t:]...)
	}
	return
}
