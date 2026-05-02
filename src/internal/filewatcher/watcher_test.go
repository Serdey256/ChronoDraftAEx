package filewatcher

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatcherStartStop(t *testing.T) {
	dir := t.TempDir()
	w := NewWatcher(dir, func() {})

	if w.IsRunning() {
		t.Fatal("expected not running before Start")
	}

	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !w.IsRunning() {
		t.Fatal("expected running after Start")
	}

	if err := w.Start(); err == nil {
		t.Fatal("expected error on double Start")
	}

	w.Stop()
	if w.IsRunning() {
		t.Fatal("expected not running after Stop")
	}

	// Stop 幂等
	w.Stop()
}

func TestDebounce(t *testing.T) {
	dir := t.TempDir()

	var count atomic.Int32
	w := NewWatcher(dir, func() {
		count.Add(1)
	})
	w.SetDebounce(200 * time.Millisecond)

	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer w.Stop()

	// 等待 watcher goroutine 就绪
	time.Sleep(50 * time.Millisecond)

	// 快速连续写入多个文件
	for i := 0; i < 5; i++ {
		f := filepath.Join(dir, "test.txt")
		os.WriteFile(f, []byte("content"), 0644)
		time.Sleep(30 * time.Millisecond)
	}

	// 等待防抖触发
	time.Sleep(500 * time.Millisecond)

	got := count.Load()
	if got != 1 {
		t.Fatalf("expected 1 callback, got %d", got)
	}
}

func TestIgnoreDir(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0755)

	var mu sync.Mutex
	var called bool
	w := NewWatcher(dir, func() {
		mu.Lock()
		called = true
		mu.Unlock()
	})
	w.SetDebounce(100 * time.Millisecond)

	if err := w.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer w.Stop()

	time.Sleep(50 * time.Millisecond)

	// 在 .git 目录下写文件
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644)

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if called {
		t.Fatal("callback should not fire for .git changes")
	}
}
