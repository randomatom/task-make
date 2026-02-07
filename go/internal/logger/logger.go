package logger

import (
	"fmt"
	"os"
	"time"
)

const yellow = "\x1b[0;33m"
const red = "\x1b[0;31m"
const reset = "\x1b[0m"

func timeStamp() string {
	now := time.Now()
	return fmt.Sprintf("[%02d:%02d:%02d] ",
		now.Hour(), now.Minute(), now.Second())
}

func Log(s string) {
	fmt.Println(s)
}

func Logd(s string) {
	fmt.Println(s)
}

func Logi(s string) {
	fmt.Println(yellow + s + reset)
}

func Loge(s string) {
	fmt.Println(red + s + reset)
}

func Logi2(s string) {
	fmt.Println(yellow + timeStamp() + s + reset)
}

func Loge2(s string) {
	fmt.Println(red + timeStamp() + s + reset)
}

func LogObj(v interface{}) {
	fmt.Printf("%#v\n", v)
}

func Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func Println(a ...interface{}) {
	fmt.Println(a...)
}

func Fatalf(format string, args ...interface{}) {
	fmt.Printf(red+format+reset, args...)
	os.Exit(1)
}

func Logf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
