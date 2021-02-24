package asyncLog

import (
	"encoding/json"
	"fmt"

	//"github.com/go-spew/spew"
	"log"
	"os"
	"path/filepath"
	"runtime"

	nsema "github.com/toolkits/concurrent/semaphore"

	//"strings"
	nsync "sync"
	"time"
)

type asyncLogType struct {
	files map[string]*LogFile // 下标是文件名
	// 避免并发new对象
	nsync.RWMutex
}
type LogFile struct {
	filename string // 原始文件名（包含完整目录）
	flag     int    // 默认为log.LstdFlags
	// 日志的等级
	level   Priority
	MaxDays int64
	// 文件切割
	logRotate struct {
		rotate LogRotate   // 默认按小时切割
		file   *os.File    // 文件操作对象
		suffix string      // 切割后的文件名后缀
		mutex  nsync.Mutex // 文件名锁
	}
	// 写入并发锁
	sema *nsema.Semaphore
}
type Gcache struct {
	// 缓存
	Lf   *LogFile
	data []byte // 缓存数据
}

// log同步的状态
type syncStatus int

const (
	statusInit  syncStatus = iota // 初始状态
	statusDoing                   // 同步中
	statusDone                    // 同步已经完成
)

// 日志切割的方式
type LogRotate int

const (
	RotateHour LogRotate = iota // 按小时切割
	RotateDate                  // 按日期切割
)
const (
	// 写日志时前缀的时间格式
	// "2006-01-02T15:04:05Z07:00"
	//logTimeFormat string = time.RFC3339
	logTimeFormat string = time.RFC3339
	TimeFormat           = "2006-01-02 15:04:05"
	// 文件写入mode
	fileOpenMode = 0666
	// 文件Flag
	fileFlag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	// 换行符
	newlineStr  = "\n"
	newlineChar = '\n'
	// 缓存切片的初始容量
	cacheInitCap = 64
)

// 是否需要Flag信息
const (
	NoFlag  = 0
	StdFlag = log.LstdFlags
	//StdFlag = log.Llongfile
)

// 异步日志变量
var asyncLog *asyncLogType
var nowFunc = time.Now

//保留时间
var MaxDay int64 = 7

//缓存通道
var CacheChannel = make(chan *Gcache, 100000)

//获取时间 yangxb
func getTimeString() string {
	return time.Unix(time.Now().Unix(), 0).Format(TimeFormat)
}

//获取文件信息 yangxb
func getFileInfo(s string) (finfo string) {
	_, file, line, _ := runtime.Caller(5)
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	file = short
	finfo = fmt.Sprintf("%v %v %v", file, line, s)
	return
}
func init() {
	asyncLog = &asyncLogType{
		files: make(map[string]*LogFile),
	}
	go startFlush()
}
func startFlush() {
	for {
		Q := <-CacheChannel
		go Q.Lf.flush(Q.data)
	}
}

//创建文件夹 yangxb
func mkLogDir(filename string) (err error) {
	abs_file, err := filepath.Abs(filename)
	if err != nil {
		return
	}
	dir := filepath.Dir(abs_file)
	_, err = os.Stat(dir)
	b := err == nil || os.IsExist(err)
	if !b {
		if err = os.MkdirAll(dir, 0777); err != nil {
			if os.IsPermission(err) {
				return
			}
		}
	}
	return
}
func NewLogFile(filename string) (lf *LogFile, err error) {
	err = mkLogDir(filename)
	if err != nil {
		return
	}
	asyncLog.Lock()
	defer asyncLog.Unlock()
	if lfExist, ok := asyncLog.files[filename]; ok {
		lf = lfExist
		lf.deleteOldLog()
		return
	}
	lf = &LogFile{
		filename: filename,
		flag:     StdFlag,
		MaxDays:  MaxDay,
		sema:     nsema.NewSemaphore(1),
	}
	asyncLog.files[filename] = lf
	// 默认按小时切割文件
	//lf.logRotate.rotate = RotateHour
	// 默认按天切割文件
	lf.logRotate.rotate = RotateDate
	lf.deleteOldLog()
	return
}
func (lf *LogFile) SetFlags(flag int) {
	lf.flag = flag
}
func (lf *LogFile) SetRotate(rotate LogRotate) {
	lf.logRotate.rotate = rotate
}

// Write 写缓存
func (lf *LogFile) Write(msg string) error {
	if lf.flag == StdFlag {
		msg = getTimeString() + " " + getFileInfo(msg) + newlineStr
	} else {
		msg = msg + newlineStr
	}
	lf.appendCache([]byte(msg))
	return nil
}

// WriteJson 写入json数据
func (lf *LogFile) WriteJson(data interface{}) error {
	bts, err := json.Marshal(data)
	if err != nil {
		return err
	}
	bts = append(bts, newlineChar)
	lf.appendCache(bts)
	return nil
}

//*********************** 以下是私有函数 ************************************
func (lf *LogFile) appendCache(msg []byte) {
	S := new(Gcache)
	S.Lf = lf
	S.data = msg
	CacheChannel <- S
}

// 同步缓存到文件中
func (lf *LogFile) flush(msg []byte) error {
	lf.sema.Acquire()
	defer lf.sema.Release()
	// 写入log文件
	file, err := lf.openFile()
	if err != nil {
		//重试
		mkLogDir(lf.filename)
		file, err = lf.openFile()
		if err != nil {
			return err
		}
	}
	_, err = file.WriteString(string(msg))
	if err != nil {
		// 重试
		_, err = file.WriteString(string(msg))
		if err != nil {
			return err
		}
	}
	return nil
}

// 获取文件名的后缀
func (lf *LogFile) getFilenameSuffix() string {
	if lf.logRotate.rotate == RotateDate {
		return nowFunc().Format("20060102")
	}
	return nowFunc().Format("2006010215")
}

//获取超时文件名后缀
func (lf *LogFile) getTimeOutFilenameSuffix() string {
	if lf.logRotate.rotate == RotateDate {
		return time.Unix(nowFunc().Unix()-(lf.MaxDays*3600*24), 0).Format("20060102")
	}
	return time.Unix(nowFunc().Unix()-43200, 0).Format("2006010215")
}
func createFile(logFilename string) (file *os.File, err error) {
	file, err = os.OpenFile(logFilename, fileFlag, fileOpenMode)
	if err != nil {
		// 重试
		file, err = os.OpenFile(logFilename, fileFlag, fileOpenMode)
		if err != nil {
			return
		}
	}
	return
}

// 打开日志文件
func (lf *LogFile) openFile() (file *os.File, err error) {
	suffix := lf.getFilenameSuffix()
	lf.logRotate.mutex.Lock()
	defer lf.logRotate.mutex.Unlock()
	logFilename := lf.filename + "." + suffix
	if suffix == lf.logRotate.suffix {
		_, errFile := os.Stat(logFilename)
		if errFile != nil && os.IsNotExist(errFile) {
			file, err = createFile(logFilename)
			if err != nil {
				return
			}
		} else {
			file = lf.logRotate.file
			return
		}
	} else {
		file, err = createFile(logFilename)
		if err != nil {
			return
		}
	}
	// 关闭旧的文件
	if lf.logRotate.file != nil {
		lf.logRotate.file.Close()
		lf.deleteOldLog()
	}
	lf.logRotate.file = file
	lf.logRotate.suffix = suffix
	return
}

// 打开日志文件(不缓存句柄)
func (lf *LogFile) openFileNoCache() (file *os.File, err error) {
	logFilename := lf.filename + "." + lf.getFilenameSuffix()
	lf.logRotate.mutex.Lock()
	defer lf.logRotate.mutex.Unlock()
	file, err = os.OpenFile(logFilename, fileFlag, fileOpenMode)
	if err != nil {
		// 重试
		file, err = os.OpenFile(logFilename, fileFlag, fileOpenMode)
		if err != nil {
			return
		}
	}
	return
}
func (lf *LogFile) deleteOldLog() {
	timeoutSuffix := lf.getTimeOutFilenameSuffix()
	timeoutFile := lf.filename + "." + timeoutSuffix
	os.Remove(timeoutFile)
}
