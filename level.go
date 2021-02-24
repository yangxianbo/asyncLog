package asyncLog

import (
	"errors"
	"fmt"

	//"github.com/go-spew/spew"
	"strings"
	"time"
)

// 日志优先级
type Priority int

const (
	LevelAll Priority = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelOff
)

var (
	// 日志等级
	levelTitle = map[Priority]string{
		LevelDebug: "[DEBUG]",
		LevelInfo:  "[INFO]",
		LevelWarn:  "[WARN]",
		LevelError: "[ERROR]",
		LevelFatal: "[FATAL]",
	}
)

func InterfaceToString(pval interface{}) string {
	var s_txt string
	switch v := (pval).(type) {
	case nil:
		s_txt = ""
	case time.Time:
		s_txt = v.Format(TimeFormat)
	case int, int8, int16, int32, int64, float32, float64, byte:
		s_txt = fmt.Sprint(v)
	case []byte:
		s_txt = string(v)
	case bool:
		if v {
			s_txt = "1"
		} else {
			s_txt = "0"
		}
	case error:
		s_txt = v.Error()
	default:
		s_txt = fmt.Sprint(v)
	}
	return s_txt
}

// NewLevelLog 写入等级日志
// 级别高于logLevel才会被写入
func NewLevelLog(filename string, logLevel Priority) (lf *LogFile, err error) {
	err = mkLogDir(filename)
	if err != nil {
		return
	}
	lf, err = NewLogFile(filename)
	if err != nil {
		return
	}
	lf.level = logLevel
	return
}
func getFormatString(msg []interface{}) (msg0 string, err error) {
	if len(msg) == 0 {
		err = errors.New("args is not enough")
		return
	}
	msg0 = InterfaceToString(msg[0])
	return
}
func (lf *LogFile) SetLevel(logLevel Priority) {
	lf.level = logLevel
}
func (lf *LogFile) Debug(msg []interface{}) error {
	return lf.writeLevelMsg(LevelDebug, msg)
}
func (lf *LogFile) Debugf(msg []interface{}) error {
	msg0, err := getFormatString(msg)
	if err != nil {
		return err
	}
	smsg := make([]interface{}, 1)
	smsg = append(smsg, fmt.Sprintf(msg0, msg[1:]...))
	return lf.writeLevelMsg(LevelDebug, smsg)
}
func (lf *LogFile) Info(msg []interface{}) error {
	return lf.writeLevelMsg(LevelInfo, msg)
}
func (lf *LogFile) Infof(msg []interface{}) error {
	msg0, err := getFormatString(msg)
	if err != nil {
		return err
	}
	smsg := make([]interface{}, 1)
	smsg = append(smsg, fmt.Sprintf(msg0, msg[1:]...))
	return lf.writeLevelMsg(LevelInfo, smsg)
}
func (lf *LogFile) Warn(msg []interface{}) error {
	return lf.writeLevelMsg(LevelWarn, msg)
}
func (lf *LogFile) Warnf(msg []interface{}) error {
	msg0, err := getFormatString(msg)
	if err != nil {
		return err
	}
	smsg := make([]interface{}, 1)
	smsg = append(smsg, fmt.Sprintf(msg0, msg[1:]...))
	return lf.writeLevelMsg(LevelWarn, smsg)
}
func (lf *LogFile) Error(msg []interface{}) error {
	return lf.writeLevelMsg(LevelError, msg)
}
func (lf *LogFile) Errorf(msg []interface{}) error {
	msg0, err := getFormatString(msg)
	if err != nil {
		return err
	}
	smsg := make([]interface{}, 1)
	smsg = append(smsg, fmt.Sprintf(msg0, msg[1:]...))
	return lf.writeLevelMsg(LevelError, smsg)
}
func (lf *LogFile) Fatal(msg []interface{}) error {
	return lf.writeLevelMsg(LevelFatal, msg)
}
func (lf *LogFile) Fatalf(msg []interface{}) error {
	msg0, err := getFormatString(msg)
	if err != nil {
		return err
	}
	smsg := make([]interface{}, 1)
	smsg = append(smsg, fmt.Sprintf(msg0, msg[1:]...))
	return lf.writeLevelMsg(LevelFatal, smsg)
}
func (lf *LogFile) writeLevelMsg(level Priority, msg []interface{}) error {
	smsgArray := []string{}
	for _, single_msg := range msg {
		str := InterfaceToString(single_msg)
		if str != "" {
			smsgArray = append(smsgArray, str)
		}
	}
	smsg := strings.Join(smsgArray, " ")
	if level >= lf.level {
		return lf.Write(levelTitle[level] + " " + smsg)
	}
	return nil
}
