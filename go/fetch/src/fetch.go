package main

import (
	"changelog"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"github.com/qiniu/api/auth/digest"
	"github.com/qiniu/api/rs"
	"github.com/qiniu/log"
	"namediff"
	"net/url"
	"qbox.us/cc"
	"qbox.us/cc/config"
	"strings"
)

var (
	ErrInvalidConf = errors.New("Invalid Config File")
)

type Task struct {
	SyncSrc string `json:"src"`
	Ak      string `json:"ak"`
	Sk      string `json:"sk"`
	Bucket  string `json:"bucket"`
	Style   string `json:"style"`
}

func (t *Task) Isvalid() bool {
	if t.SyncSrc == "" || t.Ak == "" || t.Sk == "" || t.Bucket == "" {
		return false
	}
	return true
}

func main() {

	flag.Parse()

	if flag.NArg() < 1 {
		help()
		return
	}

	var task Task
	var taskFile = flag.Arg(0)
	err := config.LoadEx(&task, taskFile)
	if err != nil || !task.Isvalid() {
		log.Warn(err, "config.Load failed:", taskFile)
		return
	}

	task.Run()
}

func help() {

	fmt.Print(`
        Usage: fetch <CfgFile>

        CfgFile is json format file. Here is an example:

        {
            "src": "/home/your/urls_file_path",
            "ak": "<SecretKey>",
            "sk": "<SecretKey>",
            "bucket": "<your_bucket>",
            "style": "<style>"
        }`)

}

func (t *Task) Run() {
	urls, err := UrlsToSync(t.SyncSrc)
	if err != nil {
		log.Error("Read Urls Failed:", err)
		return
	}

	cl, base, err := t.Synced()
	if err != nil {
		log.Error("Read Log Failed", err)
		return
	}
	left := namediff.Diff(urls, base)
	mac := digest.Mac{
		AccessKey: t.Ak,
		SecretKey: []byte(t.Sk),
	}
	log.Info("Entry Fetched:", len(base))
	log.Info("Entry ALL:", len(urls))
	log.Info("Entry TO Fetch:", len(left))
	c := rs.New(&mac)

	failed := 0
	for i := 0; i < len(left); i++ {
		urlSrc := t.srcUrl(left[i])
		key := destKey(left[i])
		err := c.Fetch(nil, urlSrc, t.Bucket, key)
		if err != nil {
			log.Error("Fetch Url Failed:", err, urlSrc)
			failed++
			continue
		}
		cl.Put(left[i])
		log.Info(urlSrc, "==>", t.Bucket, key)
	}

	if failed == 0 {
		log.Info("Fetch Success")
	} else {
		log.Info(failed, "Files Failed")
	}

	return
}

func (t *Task) srcUrl(u string) string {
	return u + t.Style
}

func destKey(u string) string {
	uparsed, _ := url.Parse(u)
	path := uparsed.Path
	if strings.HasPrefix(path, "/") {
		if len(path) > 1 {
			path = path[1:]
		} else {
			path = ""
		}
	}

	return path
}

func UrlsToSync(src string) (urls []string, err error) {
	urls, err = changelog.LoadUrls(src)
	return
}

func (t *Task) Synced() (cl changelog.Logger, urls []string, err error) {
	dir, err := cc.GetConfigDir("fetch/")
	if err != nil {
		log.Warn(err, "GetConfigDir failed")
		return
	}

	pos := strings.LastIndex(t.SyncSrc, "/")
	if pos == -1 {
		pos = 0
	}
	eigen := t.SyncSrc[pos:] + t.Bucket + t.Style
	cl, urls, err = OpenChangeLog(dir, eigen)
	return
}

func OpenChangeLog(stateDir, eigen string) (cl changelog.Logger, urls []string, err error) {

	h := md5.New()
	h.Write([]byte(eigen))
	hash := h.Sum([]byte{'2', '0'})

	clname := stateDir + base64.URLEncoding.EncodeToString(hash) + ".log"

	log.Info("Processing file:", clname)
	cl, urls, err = changelog.Open(clname)

	return
}

// -----------------------------------------------
