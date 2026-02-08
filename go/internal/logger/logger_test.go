package logger

import (
	"strings"
	"testing"
)

func TestLog(t *testing.T) {
	content := "test message"
	Log(content)
}

func TestLogd(t *testing.T) {
	content := "debug message"
	Logd(content)
}

func TestLogi(t *testing.T) {
	content := "info message"
	Logi(content)
}

func TestLoge(t *testing.T) {
	content := "error message"
	Loge(content)
}

func TestLogObj(t *testing.T) {
	testObj := map[string]int{"key": 42}
	LogObj(testObj)
}

func TestPrintf(t *testing.T) {
	Printf("test %s", "message")
}

func TestPrintln(t *testing.T) {
	Println("a", "b", "c")
}

func TestFatalf(t *testing.T) {
	// Fatalf calls os.Exit, which exits the test
	// We can't test it directly in Go testing
	// Just verify the function exists and would exit
	t.Skip("Fatalf calls os.Exit, cannot test in Go")
}

func TestLogf(t *testing.T) {
	Logf("log %s", "message")
}

func TestTimeStamp(t *testing.T) {
	ts := timeStamp()
	if !strings.HasPrefix(ts, "[") {
		t.Errorf("timeStamp() = %q, want to start with '['", ts)
	}
	if !strings.HasSuffix(ts, "] ") {
		t.Errorf("timeStamp() = %q, want to end with '] '", ts)
	}
	// Format: [HH:MM:SS]  = 10 chars
	// But with actual time it could be different, just check it contains proper format
	if len(ts) < 10 || len(ts) > 12 {
		t.Errorf("timeStamp() length = %d, expected 10-12", len(ts))
	}
}

func TestLogi2(t *testing.T) {
	Logi2("message with timestamp")
}

func TestLoge2(t *testing.T) {
	Loge2("error with timestamp")
}
