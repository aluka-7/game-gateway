// Copyright©,2020-2025
// Version: 1.0.0
// Date: 2020/8/6 10:48
// Description：日志操作类

package logger

import (
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"strings"
	"time"
)

var Log *zap.SugaredLogger

func init() {
	//设置一些基本日志格式
	encoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.CapitalLevelEncoder,
		TimeKey:     "ts",
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},
		CallerKey:    "file",
		EncodeCaller: zapcore.ShortCallerEncoder,
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},
	})
	//debug level 日志记录
	debugLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.DebugLevel
	})
	//info level 日志记录
	infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.InfoLevel
	})
	//error level 日志记录
	errorLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.ErrorLevel
	})

	//sql error 日志记录
	sqlLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.WarnLevel
	})

	var LogDebugPath = "./logs/debug/debug.log"
	var LogInfoPath = "./logs/info/info.log"
	var LogErrorPath = "./logs/error/error.log"
	var LogSqlPath = "./logs/sql/sql.log"

	// 获取 info、error日志文件的io.Writer 抽象 getWriter() 在下方实现
	debugWriter := getWriter(LogDebugPath)
	infoWriter := getWriter(LogInfoPath)
	errorWriter := getWriter(LogErrorPath)
	sqlWriter := getWriter(LogSqlPath)

	var syncerDebug zapcore.WriteSyncer
	var syncerInfo zapcore.WriteSyncer
	var syncerError zapcore.WriteSyncer
	var syncerSql zapcore.WriteSyncer
	//if beego.BConfig.RunMode == "dev" || beego.BConfig.RunMode == "test" {
	syncerDebug = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(debugWriter))
	syncerInfo = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(infoWriter))
	syncerError = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(errorWriter))
	syncerSql = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(sqlWriter))
	//} else {
	//	syncerDebug = zapcore.AddSync(debugWriter)
	//	syncerInfo = zapcore.AddSync(infoWriter)
	//	syncerError = zapcore.AddSync(errorWriter)
	//	syncerSql = zapcore.AddSync(sqlWriter)
	//}

	// 最后创建具体的Logger0
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, syncerDebug, debugLevel),
		zapcore.NewCore(encoder, syncerInfo, infoLevel),
		zapcore.NewCore(encoder, syncerError, errorLevel),
		zapcore.NewCore(encoder, syncerSql, sqlLevel),
	)
	// 需要传入 zap.AddCaller() 才会显示打日志点的文件名和行数, 有点小坑
	log := zap.New(core, zap.AddCaller())
	Log = log.Sugar()

}

func getWriter(filename string) io.Writer {
	hook, err := rotatelogs.New(
		// 生成rotatelogs的Logger 实际生成的文件名 demo.log.YYmmddHH
		// demo.log是指向最新日志的链接
		// 保存7天内的日志，每1小时(整点)分割一次日志
		strings.Replace(filename, ".log", "", -1)+"-%Y%m%d%H.log", // 没有使用go风格反人类的format格式
		//rotatelogs.WithLinkName(filename),
		//保留旧文件的最大天数
		rotatelogs.WithMaxAge(time.Hour*24*7),
		//文件轮换之间的间隔。默认情况下，日志每86400秒旋转一次
		//rotatelogs.WithRotationTime(time.Hour),
		//文件数应保留。默认情况下，此选项处于禁用状态
		//rotatelogs.WithRotationCount()
	)
	if err != nil {
		panic(err)
	}
	return hook
}
