package stat

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// EventHandler 文件事件处理函数。
type EventHandler func(path string, op fsnotify.Op)

// Watcher 文件热更新监听器。
// 功能：
// 1. 监听文件或目录变化
// 2. 自动删除 Cache 中对应路径缓存
// 3. 支持业务回调
type Watcher struct {
	mu sync.RWMutex

	watcher *fsnotify.Watcher
	cache   *Cache

	// watchedFiles 记录“文件路径 -> 父目录”
	watchedFiles map[string]string

	// watchedDirs 记录当前已监听目录
	watchedDirs map[string]struct{}

	// handlers 事件回调
	handlers []EventHandler

	closed bool
	done   chan struct{}
}

// NewWatcher 创建监听器。
func NewWatcher(cache *Cache) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	fw := &Watcher{
		watcher:      w,
		cache:        cache,
		watchedFiles: make(map[string]string, 64),
		watchedDirs:  make(map[string]struct{}, 16),
		handlers:     make([]EventHandler, 0, 4),
		done:         make(chan struct{}),
	}

	go fw.loop()
	return fw, nil
}

// AddHandler 注册事件回调。
func (fw *Watcher) AddHandler(handler EventHandler) {
	if handler == nil {
		return
	}

	fw.mu.Lock()
	fw.handlers = append(fw.handlers, handler)
	fw.mu.Unlock()
}

// Watch 监听路径。
// 传入文件：监听其父目录，并过滤该文件事件。
// 传入目录：直接监听该目录。
func (fw *Watcher) Watch(path string) error {
	if fw == nil || fw.watcher == nil {
		return errors.New("watcher is nil")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return errors.New("watcher is closed")
	}

	if info.IsDir() {
		if _, ok := fw.watchedDirs[absPath]; ok {
			return nil
		}
		if err := fw.watcher.Add(absPath); err != nil {
			return err
		}
		fw.watchedDirs[absPath] = struct{}{}
		return nil
	}

	dir := filepath.Dir(absPath)
	if _, ok := fw.watchedDirs[dir]; !ok {
		if err := fw.watcher.Add(dir); err != nil {
			return err
		}
		fw.watchedDirs[dir] = struct{}{}
	}
	fw.watchedFiles[absPath] = dir

	return nil
}

// Unwatch 取消监听路径。
func (fw *Watcher) Unwatch(path string) error {
	if fw == nil || fw.watcher == nil {
		return errors.New("watcher is nil")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return errors.New("watcher is closed")
	}

	// 如果是文件，只移除文件映射，不直接移除目录监听
	// 避免同目录下其他文件还在监听时误删目录 watcher。
	if _, ok := fw.watchedFiles[absPath]; ok {
		delete(fw.watchedFiles, absPath)
		return nil
	}

	// 如果是目录，则真正移除 watcher
	if _, ok := fw.watchedDirs[absPath]; ok {
		if err := fw.watcher.Remove(absPath); err != nil {
			// 某些平台/场景 remove 失败，不阻断内部状态清理
		}
		delete(fw.watchedDirs, absPath)

		// 顺带清理属于该目录的文件映射
		for file, dir := range fw.watchedFiles {
			if dir == absPath {
				delete(fw.watchedFiles, file)
			}
		}
	}

	return nil
}

// Close 关闭监听器。
func (fw *Watcher) Close() error {
	if fw == nil || fw.watcher == nil {
		return nil
	}

	fw.mu.Lock()
	if fw.closed {
		fw.mu.Unlock()
		return nil
	}
	fw.closed = true
	close(fw.done)
	fw.mu.Unlock()

	return fw.watcher.Close()
}

func (fw *Watcher) loop() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)

		case _, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			// 这里先吞掉错误；如果你想暴露错误通道，也可以扩展一个 OnError

		case <-fw.done:
			return
		}
	}
}

func (fw *Watcher) handleEvent(event fsnotify.Event) {
	path, err := filepath.Abs(event.Name)
	if err != nil {
		path = event.Name
	}

	fw.mu.RLock()
	cache := fw.cache
	handlers := append([]EventHandler(nil), fw.handlers...)
	_, fileWatched := fw.watchedFiles[path]
	dirWatched := fw.isPathUnderWatchedDirLocked(path)
	fw.mu.RUnlock()

	// 如果是明确监听的文件，或者路径位于被监听目录下，就处理
	if !fileWatched && !dirWatched {
		return
	}

	if cache != nil {
		cache.Delete(path)
	}

	// 有些编辑器保存文件时会用 rename/replace 策略，
	// 如果监听的是单文件，文件被替换后可以尝试重新登记。
	if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		fw.tryRewatchFile(path)
	}

	for _, handler := range handlers {
		handler(path, event.Op)
	}
}

func (fw *Watcher) isPathUnderWatchedDirLocked(path string) bool {
	for dir := range fw.watchedDirs {
		if path == dir || isSubPath(path, dir) {
			// 如果是目录监听模式，直接匹配
			// 如果是文件监听模式，因为父目录也在 watchedDirs 中，这里也可能命中；
			// 后续仍由业务决定是否关心该事件。
			return true
		}
	}
	return false
}

func (fw *Watcher) tryRewatchFile(path string) {
	fw.mu.RLock()
	_, ok := fw.watchedFiles[path]
	fw.mu.RUnlock()
	if !ok {
		return
	}

	// 文件监听本质上是监听父目录，所以通常不需要重新 Add 文件本身。
	// 这里只保留扩展点，便于后续增强。
}

func isSubPath(path, dir string) bool {
	if path == dir {
		return true
	}

	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
