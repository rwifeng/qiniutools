package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/qiniu/api/conf"
	"github.com/qiniu/log"
	"github.com/qiniu/xlog"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"qbox.us/api/v2/rs"
	"qbox.us/api/v2/rsf"
	"qbox.us/digest_auth"
	"strconv"
	"time"
)

type Config struct {
	AccessKey   string `json:"access_key"`
	SecretKey   string `json:"secret_key"`
	Bucket      string `json:"bucket"`
	StartTime   int64  `json:"start_time"`
	MaxSize     int64  `json:"max_size"`
	Prefix      string `json:"prefix"` //通配符
	EncodeFname int    `json:"encode_fname"`
	IoAddr      string `json:"io_addr"`
	IoHost      string `json:"io_host"`
	RsHost      string `json:"rs_host"`
	RsfHost     string `json:"rsf_host"`
}

type Pos struct {
	Marker string `json:"marker"`
	KeyPos int    `json:"key_pos"`
}

type Rsf struct {
	*Config
	baseDir string
	rsfs    *rsf.Service
	rss     rs.Service
}

const (
	BLOCK_BITS = 22 // Indicate that the blocksize is 4M
)

const (
	POS_INIT              = -1
	POS_KEYS_COMPLETE     = -2
	POS_KEYS_NOT_COMPLETE = -3
)
const MAX_LIMIT = 1000 //每次获取文件列表的最大限制数
const RETRY_TIMES = 3  //重试次数
const SLEEP_TIME = 2000 * time.Millisecond

func NewRsf(baseDir string) (*Rsf, error) {
	// load config
	conf := Config{}
	err := loadJsonFile(&conf, baseDir+"/qrsb.conf")
	if err != nil {
		log.Error("load qrsb.conf file error!")
		return nil, err
	}
	log.Info("use conf of:", conf)
	t := digest_auth.NewTransport(conf.AccessKey, conf.SecretKey, nil)
	rsfs := rsf.New(t, conf.RsfHost)
	RS_HOST = conf.RsHost
	rss := rs.New(t)
	return &Rsf{&conf, baseDir, rsfs, rss}, nil
}

func loadJsonFile(i interface{}, file string) error {
	d, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error("load json file", file, "err:", err)
		return err
	}
	err = json.Unmarshal(d, i)
	if err != nil {
		log.Error("unmarshal json file", file, "err:", err)
		return err
	}
	return nil
}

func saveJsonFile(i interface{}, file string) error {
	b, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		log.Error("marshal json file", file, "err:", err)
		return err
	}
	err = ioutil.WriteFile(file+".swap", b, 0600)
	if err != nil {
		log.Error("save json file", file+".swap", "err:", err)
		return err
	}

	err = os.Remove(file)
	if err != nil {
		log.Error("remove json file err:", file)
	}
	err = os.Rename(file+".swap", file)
	if err != nil {
		log.Error("rename json file err:", file)
	}
	return err
}

func (p *Rsf) saveKeyPath(xl *xlog.Logger, rawKey, key string, r io.Reader) error {
	filename := p.baseDir + "/data/" + rawKey
	dir := path.Dir(filename)
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		log.Info("saveKeyPath os.MkdirAll fail:", err)
		return p.saveKey(xl, key, r)
	}
	f, err := os.Create(filename)
	if err != nil {
		log.Info("saveKeyPath os.Create fail:", err)
		return p.saveKey(xl, key, r)
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	log.Info("save key (path) done:", rawKey, err)
	return err
}

func (p *Rsf) saveKey(xl *xlog.Logger, key string, r io.Reader) error {
	h := sha1.New()
	io.WriteString(h, key)
	s := hex.EncodeToString(h.Sum(nil))
	dir := p.baseDir + "/data/" + s[:2] + "/" + s[2:4]
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		return err
	}
	if key == "" {
		key = "_empty"
	}
	f, err := os.Create(dir + "/" + key)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	xl.Info("save key done:", key, err)
	return err
}

//下载文件item.Name
//retry:重试次数
func (p *Rsf) download(item rsf.DumpItem, retry int) (err error) {
	log.Info("downloading file :", item.Name, " ,file size = ", item.Fsize, "bytes")
	for i := 1; i <= retry; i++ {
		code, err := p.getFile(item.Name)
		if err == nil {
			log.Info("download completed!")
			return nil
		}
		log.Error("getFile err:", code, err)
		if code == 612 {
			break
		}
		log.Error("retry download ", item.Name, " ", i, " times")
		time.Sleep(SLEEP_TIME)

	}
	//重试retry次后，下载失败
	ffail, err := os.OpenFile(p.baseDir+"/qrsb.failkeys", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
	if err == nil {
		defer ffail.Close()
		b, err := json.Marshal(item)
		if err == nil {
			ffail.WriteString(string(b))
		}
	} else {
		log.Error("Open qrsb.failkeys err")
		return err
	}
	return
}

func (p *Rsf) HttpGet(fn string) (*http.Response, error) {
	url := p.IoAddr + "/" + fn
	req, err := http.NewRequest("Get", url, nil)
	if err != nil {
		return nil, err
	}

	req.Host = p.IoHost
	client := &http.Client{}

	return client.Do(req)
}

func (p *Rsf) getFile(filename string) (int, error) {
	key := base64.URLEncoding.EncodeToString([]byte(filename))
	// key2 := p.Bucket + ":" + string(filename)
	// data, code, err := p.rss.Get(key2, "")

	// if err != nil {
	// 	log.Error("rss.Get err:", code, err)
	// 	return code, err
	// }
	// resp, err := http.Get(data.URL)
	resp, err := p.HttpGet(filename)
	if err != nil {
		log.Error("get io url err:", err, filename)
		return 0, err
	}
	defer resp.Body.Close()
	xl := xlog.NewWith(resp.Header.Get("X-Reqid"))
	if resp.StatusCode != 200 {
		xl.Error("get io url code != 200:", resp.StatusCode)
		return 0, errors.New("get io url err:" + strconv.Itoa(resp.StatusCode))
	}
	if p.EncodeFname == 0 {
		return 0, p.saveKeyPath(xl, filename, key, resp.Body)
	}
	return 0, p.saveKey(xl, key, resp.Body)
}

// To get how many blocks does the file has.
func BlockCount(fsize int64) int {
	blockMask := int64((1 << BLOCK_BITS) - 1)
	return int((fsize + blockMask) >> BLOCK_BITS)
}

func CalSha1(r io.Reader) (sha1Code []byte, err error) {
	h := sha1.New()
	_, err = io.Copy(h, r)
	if err != nil {
		return
	}
	sha1Code = h.Sum(nil)
	return
}
func GetEtag(filename string) (etag string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return
	}
	fsize := fi.Size()
	blockCnt := BlockCount(fsize)
	sha1Buf := make([]byte, 0, 21)

	var sha1Code []byte
	if blockCnt <= 1 { // file size <= 4M
		sha1Buf = append(sha1Buf, 0x16)
		sha1Code, err = CalSha1(f)
		if err != nil {
			return
		}
		sha1Buf = append(sha1Buf, sha1Code...)
	} else { // file size > 4M
		sha1Buf = append(sha1Buf, 0x96)
		sha1BlockBuf := make([]byte, 0, blockCnt*20)

		for i := 0; i < blockCnt; i++ {
			body := io.LimitReader(f, 1<<BLOCK_BITS)
			sha1Code, err = CalSha1(body)
			if err != nil {
				return
			}
			sha1BlockBuf = append(sha1BlockBuf, sha1Code...)
		}
		tmpBuf := bytes.NewBuffer(sha1BlockBuf)
		var sha1Final []byte
		sha1Final, err = CalSha1(tmpBuf)
		if err != nil {
			return
		}
		sha1Buf = append(sha1Buf, sha1Final...)
	}
	etag = base64.URLEncoding.EncodeToString(sha1Buf)
	return
}

//是否需要重新下载文件
//判断准则：文件名->文件大小->文件HASH
func (p *Rsf) isNeedReload(item rsf.DumpItem, pos *Pos) bool {
	if item.Time <= p.Config.StartTime {
		return false
	}
	var path string
	if p.Config.EncodeFname == 0 {
		path = p.baseDir + "/data/" + item.Name
	} else {
		key := base64.URLEncoding.EncodeToString([]byte(item.Name))
		h := sha1.New()
		io.WriteString(h, key)
		s := hex.EncodeToString(h.Sum(nil))
		dir := p.baseDir + "/data/" + s[:2] + "/" + s[2:4]
		path = dir + "/" + key
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != item.Fsize {
		return true
	}
	etag, err := GetEtag(path)
	if err != nil {
		log.Error("GET Etag err:", item)
		return true
	}
	return etag != item.Hash
}

func (p *Rsf) isFirstRun() bool {
	_, err := os.Stat(p.baseDir + "/qrsb.pos")
	return err != nil
}

func (p *Rsf) firstRun() error {
	log.Info("first run, init...")
	pos := Pos{"", POS_INIT}
	err := saveJsonFile(&pos, p.baseDir+"/qrsb.pos")
	return err
}

//获取文件列表并下载文件
//1）获取列表
//2)依次下载列表中每一项对应的文件
//bool:true,fetch 结束
func (p *Rsf) fetch(pos *Pos) (bool, error) {
	log.Info("fetching, marker=", pos.Marker)
	xl := xlog.NewDummy()
	ret, err := p.rsfs.ListPrefix(xl, p.Bucket, p.Config.Prefix, pos.Marker, MAX_LIMIT)
	if err != nil {
		xl.Error("fetch err:", err, pos, p.Bucket)
		return false, err
	}
	if len(ret.Items) == 0 {
		return true, nil
	}
	for _, i := range ret.Items {
		if p.MaxSize <= 0 || i.Fsize <= p.MaxSize {
			if p.isNeedReload(i, pos) {
				p.download(i, RETRY_TIMES)
			} else {
				log.Info(i.Name + " already exists,skip!")
			}
		}
	}
	pos.Marker = ret.Marker
	return len(ret.Items) < MAX_LIMIT, nil
}

func (p *Rsf) Run(pos *Pos) (err error) {

	done := false
	for {
		done, err = p.fetch(pos)
		if err != nil || done {
			break
		}
		saveJsonFile(pos, p.baseDir+"/qrsb.pos")
	}

	if err == nil { //所有文件备份完成
		pos.KeyPos = POS_KEYS_COMPLETE
	} else {
		pos.KeyPos = POS_KEYS_NOT_COMPLETE
	}

	saveJsonFile(pos, p.baseDir+"/qrsb.pos")

	return
}

func (p *Rsf) printResult() {
	pos := Pos{}
	err := loadJsonFile(&pos, p.baseDir+"/qrsb.pos")
	if err != nil {
		return
	}
	if pos.Marker != "" {
		fmt.Println("\nResult:download ABORT.")
	} else {
		if pos.KeyPos == POS_INIT {
			fmt.Println("\nResult: No key downloaded.")
		} else if pos.KeyPos == POS_KEYS_COMPLETE {
			fmt.Println("\nResult: All keys download sucess!")
		} else if pos.KeyPos == POS_KEYS_NOT_COMPLETE {
			fmt.Println("\nResult: Fail to save some keys, please check qrsb.failkeys. ")
		}
	}
}
func main() {
	if len(os.Args) < 2 {
		fmt.Println("qrsb <dir>")
		os.Exit(1)
	}
	p, err := NewRsf(os.Args[1])
	if err != nil {
		log.Error("err:", err)
		os.Exit(2)
	}
	if p.isFirstRun() {
		p.firstRun()
	}
	// check pos
	pos := Pos{}
	err = loadJsonFile(&pos, p.baseDir+"/qrsb.pos")
	// pos.Marker = "" //置空,重新开始
	if err != nil {
		log.Error("err:load qrsb.pos file failed,ABORT!")
		os.Exit(2)
	}
	err = p.Run(&pos)
	if err != nil {
		log.Error("err:", err)
		p.printResult()
		os.Exit(2)
	}
	log.Info("Done!")
	p.printResult()
	return
}
