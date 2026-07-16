package file

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/log/internal"
	"github.com/xbaseio/xbase/utils/xos"
	"github.com/xbaseio/xbase/utils/xtime"
	"github.com/xbaseio/xbase/xerrors"
)

const Name = "file"

const (
	gzipExt              = ".gz"
	cleanupCheckInterval = time.Hour
)

type Syncer struct {
	opts        *options
	ctx         context.Context
	cancel      context.CancelFunc
	fileDir     string
	fileName    string
	fileExt     string
	fileTag     string
	fileVersion int64
	gzipExt     string
	fileMu      sync.Mutex
	queueMu     sync.Mutex
	queueCond   *sync.Cond
	pending     []entry
	size        int64
	file        *os.File
	writer      *bufio.Writer
	closing     atomic.Bool
	wg          sync.WaitGroup
	formatter   internal.Formatter
	classified  map[internal.Level]*Syncer
}

type entry struct {
	now time.Time
	buf internal.Buffer
}

func NewSyncer(opts ...Option) *Syncer {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	s := &Syncer{}
	s.opts = o
	s.init()

	return s
}

func (s *Syncer) init() {
	if s.opts.classifiedStorage {
		s.initClassified()
		return
	}
	path, file := filepath.Split(s.opts.path)
	list := strings.Split(file, ".")
	switch c := len(list); c {
	case 1:
		s.fileName = list[0]
	default:
		s.fileName, s.fileExt = strings.Join(list[:c-1], "."), "."+list[c-1]
	}

	s.fileDir = path
	s.gzipExt = gzipExt
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.queueCond = sync.NewCond(&s.queueMu)

	if s.opts.format == FormatJson {
		s.formatter = internal.NewJsonFormatter()
	} else {
		s.formatter = internal.NewTextFormatter()
	}

	if err := s.ensureDir(); err != nil {
		return
	}

	_ = s.migrateLegacyBaseFile()
	_ = s.openCurrentFile()
	_ = s.cleanupExpiredFiles(xtime.Now())

	s.wg.Add(1)
	go s.writeLoop()
	go s.tickCleanup()
}

// Name 同步器名称
func (s *Syncer) Name() string {
	return Name
}

// Write 写入日志
func (s *Syncer) Write(entity *internal.Entity) error {
	if s.closing.Load() {
		return xerrors.ErrSyncerClosed
	}
	if len(s.classified) > 0 {
		var firstErr error
		for level, syncer := range s.classified {
			if entity.Level.Priority() >= level.Priority() {
				if err := syncer.Write(entity); err != nil && firstErr == nil {
					firstErr = err
				}
			}
		}
		return firstErr
	}

	return s.doWrite(entry{
		buf: s.formatter.Format(entity),
		now: entity.Now,
	})
}

// 执行写入日志操作
func (s *Syncer) doWrite(e entry) error {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()

	if s.closing.Load() {
		if e.buf != nil {
			e.buf.Release()
		}
		return xerrors.ErrSyncerClosed
	}

	s.pending = append(s.pending, e)
	s.queueCond.Signal()

	return nil
}

// Close 关闭同步器
func (s *Syncer) Close() error {
	if !s.closing.CompareAndSwap(false, true) {
		return xerrors.ErrSyncerClosed
	}
	if len(s.classified) > 0 {
		var firstErr error
		for _, syncer := range s.classified {
			if err := syncer.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}

	s.cancel()
	s.queueMu.Lock()
	s.queueCond.Broadcast()
	s.queueMu.Unlock()

	s.wg.Wait()

	s.fileMu.Lock()
	defer s.fileMu.Unlock()

	if s.writer != nil {
		_ = s.writer.Flush()
	}

	if s.file == nil {
		return nil
	}

	return s.file.Close()
}

func (s *Syncer) initClassified() {
	s.classified = make(map[internal.Level]*Syncer, 6)
	levels := []internal.Level{
		internal.LevelDebug, internal.LevelInfo, internal.LevelWarn,
		internal.LevelError, internal.LevelFatal, internal.LevelPanic,
	}
	for _, level := range levels {
		o := *s.opts
		o.path = classifiedPath(o.path, string(level))
		o.classifiedStorage = false
		child := &Syncer{opts: &o}
		child.init()
		s.classified[level] = child
	}
}

func classifiedPath(path, level string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + "." + level + ext
}

// 尝试将数据刷入文件中
func (s *Syncer) writeLoop() {
	defer s.wg.Done()

	for {
		items := s.takePending()
		if len(items) == 0 {
			if s.closing.Load() {
				return
			}
			continue
		}

		s.fileMu.Lock()
		for _, item := range items {
			if err := s.writeEntry(item, false); err != nil && item.buf != nil {
				item.buf.Release()
			}
		}
		_ = s.flushWriter()
		s.fileMu.Unlock()
	}
}

// 写入将缓冲区数据写入文件
func (s *Syncer) takePending() []entry {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()

	for len(s.pending) == 0 && !s.closing.Load() {
		s.queueCond.Wait()
	}

	if len(s.pending) == 0 {
		return nil
	}

	items := s.pending
	s.pending = nil
	return items
}

// 写入日志
func (s *Syncer) writeEntry(e entry, isAutoFlush bool) error {
	if s.file == nil {
		if err := s.openCurrentFile(); err != nil {
			return err
		}
	}

	nextTag := s.makeFileTag(e.now)
	if nextTag != s.fileTag {
		if err := s.flushWriter(); err != nil {
			return err
		}

		if err := s.rotateTo(nextTag, 0); err != nil {
			return err
		}
	}

	if e.buf != nil {
		size, err := s.writer.Write(e.buf.Bytes())
		e.buf.Release()
		if err != nil {
			return err
		}
		s.size += int64(size)
	}

	if isAutoFlush {
		if err := s.flushWriter(); err != nil {
			return err
		}
	}

	if s.opts.maxSize > 0 && s.size >= s.opts.maxSize {
		if err := s.flushWriter(); err != nil {
			return err
		}

		if err := s.rotateTo(s.fileTag, s.fileVersion+1); err != nil {
			return err
		}
	}

	return nil
}

func (s *Syncer) tickCleanup() {
	if s.opts.maxAge <= 0 {
		return
	}

	ticker := time.NewTicker(cleanupCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case now, ok := <-ticker.C:
			if !ok {
				return
			}

			s.fileMu.Lock()
			_ = s.cleanupExpiredFiles(now)
			s.fileMu.Unlock()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Syncer) rotateTo(fileTag string, fileVersion int64) error {
	oldPath := s.currentFilePath(s.fileTag, s.fileVersion, s.fileExt)

	if s.file != nil {
		if err := s.file.Close(); err != nil {
			return err
		}
		s.file = nil
	}

	if err := s.openFile(fileTag, fileVersion); err != nil {
		return err
	}

	if s.opts.compress && oldPath != "" && oldPath != s.currentFilePath(s.fileTag, s.fileVersion, s.fileExt) {
		gzipPath := oldPath + gzipExt

		s.wg.Go(func() {
			_ = s.compressFile(gzipPath, oldPath)
		})
	}

	_ = s.cleanupExpiredFiles(xtime.Now())

	return nil
}

// 压缩文件
func (s *Syncer) compressFile(dst, src string) (err error) {
	var (
		srcFile *os.File
		dstFile *os.File
	)

	if srcFile, err = os.Open(src); err != nil {
		return
	}

	defer func() {
		_ = srcFile.Close()

		if err == nil {
			_ = os.Remove(src)
		}
	}()

	if dstFile, err = os.Create(dst); err != nil {
		return err
	}

	defer func() {
		_ = dstFile.Close()
	}()

	dstWriter := gzip.NewWriter(dstFile)
	defer func() {
		_ = dstWriter.Close()
	}()

	if _, err = io.Copy(dstWriter, bufio.NewReader(srcFile)); err != nil {
		return
	}

	return
}

func (s *Syncer) ensureDir() error {
	if _, err := os.Stat(s.fileDir); err != nil {
		if err = os.MkdirAll(filepath.Dir(s.opts.path), 0755); err != nil {
			return err
		}
	}

	return nil
}

func (s *Syncer) openCurrentFile() error {
	tag := s.makeFileTag(xtime.Now())
	version := s.resolveOpenVersion(tag)

	return s.openFile(tag, version)
}

func (s *Syncer) openFile(fileTag string, fileVersion int64) error {
	if err := s.ensureDir(); err != nil {
		return err
	}

	filePath := s.currentFilePath(fileTag, fileVersion, s.fileExt)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	fi, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}

	s.fileTag = fileTag
	s.fileVersion = fileVersion
	s.size = fi.Size()
	s.file = file

	if s.writer == nil {
		s.writer = bufio.NewWriter(file)
	} else {
		s.writer.Reset(file)
	}

	return nil
}

func (s *Syncer) resolveOpenVersion(fileTag string) int64 {
	version := s.latestVersion(fileTag)
	if version < 0 {
		return 0
	}

	if s.opts.maxSize > 0 {
		filePath := s.currentFilePath(fileTag, version, s.fileExt)
		if fi, err := os.Stat(filePath); err == nil && fi.Size() >= s.opts.maxSize {
			return version + 1
		}
	}

	return version
}

func (s *Syncer) latestVersion(fileTag string) int64 {
	entries, err := os.ReadDir(s.fileDir)
	if err != nil {
		return -1
	}

	maxVersion := int64(-1)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		tag, version, compressed, ok := s.parseStoredFileName(entry.Name())
		if !ok || compressed || tag != fileTag {
			continue
		}

		if version > maxVersion {
			maxVersion = version
		}
	}

	return maxVersion
}

func (s *Syncer) migrateLegacyBaseFile() error {
	if s.opts.rotate == RotateNone {
		return nil
	}

	legacyPath := s.opts.path
	fi, err := xos.Stat(legacyPath)
	if err != nil {
		return nil
	}

	fileTag := s.makeFileTag(fi.CreateTime())
	if fileTag == "" {
		fileTag = s.makeFileTag(fi.ModifyTime())
	}
	if fileTag == "" {
		fileTag = s.makeFileTag(xtime.Now())
	}

	version := s.latestVersion(fileTag)
	if version < 0 {
		version = 0
	} else {
		version++
	}

	targetPath := s.currentFilePath(fileTag, version, s.fileExt)
	if targetPath == legacyPath {
		return nil
	}

	if _, err = os.Stat(targetPath); err == nil {
		return nil
	}

	return os.Rename(legacyPath, targetPath)
}

func (s *Syncer) cleanupExpiredFiles(now time.Time) error {
	if s.opts.maxAge <= 0 {
		return nil
	}

	entries, err := os.ReadDir(s.fileDir)
	if err != nil {
		return err
	}

	expireBefore := now.Add(-s.opts.maxAge)
	currentPath := s.currentFilePath(s.fileTag, s.fileVersion, s.fileExt)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if _, _, _, ok := s.parseStoredFileName(name); !ok && name != filepath.Base(s.opts.path) {
			continue
		}

		fullPath := filepath.Join(s.fileDir, name)
		if fullPath == currentPath {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().After(expireBefore) {
			continue
		}

		_ = os.Remove(fullPath)
	}

	return nil
}

func (s *Syncer) parseStoredFileName(fileName string) (fileTag string, fileVersion int64, compressed bool, ok bool) {
	body := ""

	switch {
	case fileName == s.fileName+s.fileExt:
		return "", 0, false, true
	case strings.HasSuffix(fileName, s.fileExt):
		body = strings.TrimSuffix(fileName, s.fileExt)
	case strings.HasSuffix(fileName, s.fileExt+s.gzipExt):
		body = strings.TrimSuffix(fileName, s.fileExt+s.gzipExt)
		compressed = true
	default:
		return "", 0, false, false
	}

	if !strings.HasPrefix(body, s.fileName) {
		return "", 0, false, false
	}

	body = strings.TrimPrefix(body, s.fileName)
	if body == "" {
		return "", 0, compressed, true
	}
	if !strings.HasPrefix(body, ".") {
		return "", 0, false, false
	}

	parts := strings.Split(strings.TrimPrefix(body, "."), ".")
	switch len(parts) {
	case 1:
		if version, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
			return "", version, compressed, true
		}
		return parts[0], 0, compressed, true
	case 2:
		version, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return "", 0, false, false
		}
		return parts[0], version, compressed, true
	default:
		return "", 0, false, false
	}
}

func (s *Syncer) currentFilePath(fileTag string, fileVersion int64, fileExt string) string {
	return filepath.Join(s.fileDir, s.makeFileName(fileTag, fileVersion, fileExt))
}

// 生成文件名称
func (s *Syncer) makeFileName(fileTag string, fileVersion int64, fileExt string) string {
	if fileTag == "" {
		if fileVersion <= 0 {
			return s.fileName + fileExt
		}
		return fmt.Sprintf("%s.%d%s", s.fileName, fileVersion, fileExt)
	}

	if fileVersion <= 0 {
		return fmt.Sprintf("%s.%s%s", s.fileName, fileTag, fileExt)
	}

	return fmt.Sprintf("%s.%s.%d%s", s.fileName, fileTag, fileVersion, fileExt)
}

// 生成文件标签
func (s *Syncer) makeFileTag(t time.Time) string {
	switch s.opts.rotate {
	case RotateYear:
		return t.Format("2006")
	case RotateMonth:
		return t.Format("200601")
	case RotateWeek:
		year, week := t.ISOWeek()
		return fmt.Sprintf("%d%02d", year, week)
	case RotateDay:
		return t.Format("20060102")
	case RotateHour:
		return t.Format("2006010215")
	default:
		return ""
	}
}

func (s *Syncer) flushWriter() error {
	if s.writer == nil {
		return nil
	}

	return s.writer.Flush()
}
