// Package filewatcher 基于 fsnotify 的文件系统监听器，支持防抖和目录忽略
package filewatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher 文件系统监听器，监听项目目录变化并触发回调
type Watcher struct {
	projectRoot string
	debounce    time.Duration
	ignoreList  []string
	onChange    func()
	running     bool
	mu          sync.Mutex
	watcher     *fsnotify.Watcher
	timer       *time.Timer
}

// NewWatcher 创建文件监听器
// 默认防抖 2 秒，忽略 .git、.chronodraft、node_modules、vendor 目录
func NewWatcher(projectRoot string, onChange func()) *Watcher {
	return &Watcher{
		projectRoot: projectRoot,
		debounce:    2 * time.Second,
		ignoreList:  []string{".git", ".chronodraft", "node_modules", "vendor"},
		onChange:    onChange,
	}
}

// SetDebounce 设置防抖间隔
func (w *Watcher) SetDebounce(d time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.debounce = d
}

// IsRunning 返回监听器是否正在运行
func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// Start 启动文件监听，非阻塞，内部启动 goroutine
func (w *Watcher) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return fmt.Errorf("watcher already running")
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	w.watcher = fw

	if err := w.addRecursive(w.projectRoot); err != nil {
		fw.Close()
		return fmt.Errorf("add watch paths: %w", err)
	}

	w.running = true
	go w.loop()

	return nil
}

// Stop 停止文件监听
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	w.running = false
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	if w.watcher != nil {
		w.watcher.Close()
		w.watcher = nil
	}
}

// shouldIgnore 判断路径是否包含被忽略的目录段
func (w *Watcher) shouldIgnore(path string) bool {
	for _, seg := range w.ignoreList {
		if strings.Contains(path, string(os.PathSeparator)+seg+string(os.PathSeparator)) ||
			strings.HasSuffix(path, string(os.PathSeparator)+seg) ||
			path == seg {
			return true
		}
	}
	return false
}

// addRecursive 递归添加目录监听，跳过被忽略的目录
func (w *Watcher) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 跳过无法访问的路径
		}
		if !info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(w.projectRoot, path)
		if w.shouldIgnore(rel) && rel != "." {
			return filepath.SkipDir
		}
		return w.watcher.Add(path)
	})
}

// scheduleDebounce 重置防抖定时器
func (w *Watcher) scheduleDebounce() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.debounce, func() {
		if w.onChange != nil {
			w.onChange()
		}
	})
}

// loop fsnotify 事件循环
func (w *Watcher) loop() {
	for {
		w.mu.Lock()
		running := w.running
		fw := w.watcher
		w.mu.Unlock()

		if !running || fw == nil {
			return
		}

		select {
		case event, ok := <-fw.Events:
			if !ok {
				return
			}
			if w.shouldIgnore(event.Name) {
				continue
			}
			// 新建目录时自动添加监听
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					rel, _ := filepath.Rel(w.projectRoot, event.Name)
					if !w.shouldIgnore(rel) {
						fw.Add(event.Name)
					}
				}
			}
			w.scheduleDebounce()

		case err, ok := <-fw.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "[filewatcher] error: %v\n", err)
		}
	}
}
