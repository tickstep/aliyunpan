package webdav

import (
	"context"
	"errors"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/library-go/logger"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/webdav"
)

// NoSniffFileInfo wraps any generic FileInfo interface and bypasses mime type sniffing.
type NoSniffFileInfo struct {
	os.FileInfo
}

func (w NoSniffFileInfo) ContentType(ctx context.Context) (contentType string, err error) {
	if mimeType := mime.TypeByExtension(path.Ext(w.FileInfo.Name())); mimeType != "" {
		// We can figure out the mime from the extension.
		return mimeType, nil
	} else {
		// We can't figure out the mime type without sniffing, call it an octet stream.
		return "application/octet-stream", nil
	}
}


// 文件系统
type WebDavDir struct {
	webdav.Dir
	NoSniff bool
	panClientProxy *PanClientProxy
	fileInfo WebDavFileInfo
	uploadChunkSize int
}

// sliceClean is equivalent to but slightly more efficient than
// path.Clean("/" + name).
func sliceClean(name string) string {
	if name == "" || name[0] != '/' {
		name = "/" + name
	}
	return path.Clean(name)
}

// formatAbsoluteName 将name名称更改为绝对路径
func (d WebDavDir) formatAbsoluteName(pathStr string) string {
	if strings.Index(pathStr, "/") != 0 {
		pathStr = d.fileInfo.fullPath + "/" + pathStr
	}
	return pathStr
}

func (d WebDavDir) getSessionId(ctx context.Context) string {
	v := ctx.Value(KeySessionId)
	if v != nil{
		return v.(string)
	}
	return ""
}

func (d WebDavDir) getContentLength(ctx context.Context) int64 {
	v := ctx.Value(KeyContentLength)
	if v != nil{
		return v.(int64)
	}
	return 0
}

func (d WebDavDir) getUserId(ctx context.Context) string {
	v := ctx.Value(KeyUserId)
	if v != nil{
		return v.(string)
	}
	return "anonymous"
}

func (d WebDavDir) resolve(name string) string {
	// This implementation is based on Dir.Open's code in the standard net/http package.
	if filepath.Separator != '/' && strings.IndexRune(name, filepath.Separator) >= 0 ||
		strings.Contains(name, "\x00") {
		return ""
	}
	dir := string(d.Dir)
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, filepath.FromSlash(sliceClean(name)))
}

func (d WebDavDir) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	if name = d.resolve(name); name == "" {
		return os.ErrNotExist
	}
	return d.panClientProxy.Mkdir(name, perm)
}

func (d WebDavDir) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if name = d.resolve(name); name == "" {
		return nil, os.ErrNotExist
	}
	name = d.formatAbsoluteName(d.formatAbsoluteName(name))

	logger.Verbosef("OpenFile file %s flag:\n O_RDONLY=%t\n O_WRONLY=%t\n O_RDWR=%t\n O_APPEND=%t\n O_CREATE=%t\n O_EXCL=%t\n O_SYNC=%t\n O_TRUNC=%t\n",
		name,
		flag&os.O_RDONLY == 0, flag&os.O_WRONLY != 0, flag&os.O_RDWR != 0, flag&os.O_APPEND != 0,
		flag&os.O_CREATE != 0, flag&os.O_EXCL != 0, flag&os.O_SYNC != 0, flag&os.O_TRUNC != 0)

	if name == d.fileInfo.fullPath {
		return &WebDavFile{
			panClientProxy:   d.panClientProxy,
			nameSnapshot:     d.fileInfo,
			childrenSnapshot: nil,
			listPos:          0,
			readPos:          0,
			writePos:         0,
		}, nil
	}

	if flag&(os.O_SYNC|os.O_APPEND) != 0 {
		// doesn't support these flags
		return nil, os.ErrInvalid
	}
	if flag&os.O_CREATE != 0 {
		if flag&os.O_EXCL != 0 {
			return nil, os.ErrExist
		}
		// create file instance for writing
		_, e := d.panClientProxy.UploadFilePrepare(d.getUserId(ctx), name, d.getContentLength(ctx), int64(d.uploadChunkSize))
		if e != nil {
			return nil, e
		}
	}
	if flag&(os.O_WRONLY|os.O_RDWR) != 0 && flag&os.O_TRUNC != 0 {
		// file must be created ready
		// get ready to write data to file stream
		logger.Verboseln("get ready to write data to file stream")
		fus,err2 := d.panClientProxy.UploadFileCache(d.getUserId(ctx), name)
		if err2 != nil {
			return nil, err2
		}
		return &WebDavFile{
			panClientProxy:   d.panClientProxy,
			nameSnapshot:     WebDavFileInfo{
				fileId:   fus.fileId,
				name:     path.Base(fus.filePath),
				size:     fus.fileSize,
				mode:     0,
				modTime:  time.Unix(fus.timestamp, 0),
				fullPath: fus.filePath,
			},
			childrenSnapshot: nil,
			listPos:          0,
			readPos:          0,
			writePos:         fus.fileWritePos,
			sessionId: d.getSessionId(ctx),
			userId: d.getUserId(ctx),
		}, nil
	}

	// default action, open file to read
	fileItem,e := d.panClientProxy.FileInfoByPath(name)
	if e != nil {
		logger.Verboseln("OpenFile failed, file path not existed: " + name)
		return nil, e
	}
	wdfi := NewWebDavFileInfo(fileItem)
	wdfi.fullPath = d.formatAbsoluteName(name)
	return &WebDavFile{
		panClientProxy:   d.panClientProxy,
		nameSnapshot:     wdfi,
		childrenSnapshot: nil,
		listPos:          0,
		readPos:          0,
		writePos:         0,
		sessionId: d.getSessionId(ctx),
		userId: d.getUserId(ctx),
	}, nil
}

func (d WebDavDir) RemoveAll(ctx context.Context, name string) error {
	if name = d.resolve(name); name == "" {
		return os.ErrNotExist
	}
	if name == filepath.Clean(string(d.Dir)) {
		// Prohibit removing the virtual root directory.
		return os.ErrInvalid
	}
	return d.panClientProxy.RemoveAll(name)
}

func (d WebDavDir) Rename(ctx context.Context, oldName, newName string) error {
	if oldName = d.resolve(oldName); oldName == "" {
		return os.ErrNotExist
	}
	if newName = d.resolve(newName); newName == "" {
		return os.ErrNotExist
	}
	if root := filepath.Clean(string(d.Dir)); root == oldName || root == newName {
		// Prohibit renaming from or to the virtual root directory.
		return os.ErrInvalid
	}
	if path.Dir(oldName) == path.Dir(newName) {
		// rename
		return d.panClientProxy.Rename(oldName, newName)
	} else {
		// move file
		return d.panClientProxy.Move(oldName, newName)
	}
}

func (d WebDavDir) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	f := &d.fileInfo
	if name != "" {
		fileItem,e := d.panClientProxy.FileInfoByPath(d.formatAbsoluteName(name))
		if e != nil {
			logger.Verboseln("file path not existed: " + d.formatAbsoluteName(name))
			return nil, os.ErrNotExist
		}
		*f = NewWebDavFileInfo(fileItem)
	}
	return f, nil
}





// WebDavFile 文件实例
type WebDavFile struct {
	webdav.File

	// 网盘Client
	panClientProxy *PanClientProxy

	// nameSnapshot 文件关联的网盘文件信息
	nameSnapshot WebDavFileInfo

	// childrenSnapshot 目录下的子文件信息列表
	childrenSnapshot []WebDavFileInfo

	listPos int

	// 读取偏移值
	readPos int64

	// 写入偏移值
	writePos int64

	// 会话ID
	sessionId string

	// 用户ID
	userId string
}

func (f *WebDavFile) Close() error {
	f.readPos = 0
	f.writePos = 0
	return nil
}

func (f *WebDavFile) Read(p []byte) (int, error) {
	count, err := f.panClientProxy.DownloadFilePart(f.sessionId, f.nameSnapshot.fileId, f.readPos, p)
	if err != nil {
		return 0, err
	}
	f.readPos += int64(count)
	return count, nil
}

// Readdir 获取文件目录
func (f *WebDavFile) Readdir(count int) (fis []os.FileInfo, err error) {
	if f.childrenSnapshot == nil || len(f.childrenSnapshot) == 0 {
		fileList, e := f.panClientProxy.FileListGetAll(f.nameSnapshot.fullPath)
		if e != nil {
			return nil, e
		}
		for _,fileItem := range fileList {
			wdfi := NewWebDavFileInfo(fileItem)
			wdfi.fullPath = f.nameSnapshot.fullPath + "/" + wdfi.name
			f.childrenSnapshot = append(f.childrenSnapshot, wdfi)
		}
	}

	realCount := count
	if (f.listPos + realCount) > len(f.childrenSnapshot) {
		realCount = len(f.childrenSnapshot) - f.listPos
	}
	if realCount == 0 {
		realCount = len(f.childrenSnapshot)
	}

	fis = make([]os.FileInfo, realCount)
	idx := 0
	for idx < realCount {
		fis[idx] = &f.childrenSnapshot[f.listPos + idx]
		idx ++
	}
	return fis, nil
}

func (f *WebDavFile) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
		logger.Verboseln(f.sessionId + " SeekStart offset = " + strconv.Itoa(int(offset)) + " return offset = " + strconv.Itoa(int(abs)))
	case io.SeekCurrent:
		abs = f.readPos + offset
		logger.Verboseln(f.sessionId + " SeekCurrent offset = " + strconv.Itoa(int(offset)) + " return offset = " + strconv.Itoa(int(abs)))
	case io.SeekEnd:
		abs = f.nameSnapshot.size + offset
		logger.Verboseln(f.sessionId + " SeekEnd offset = " + strconv.Itoa(int(offset)) + " return offset = " + strconv.Itoa(int(abs)))
	default:
		return 0, errors.New("invalid whence")
	}
	if abs < 0 {
		return 0, os.ErrInvalid
	}
	f.readPos = abs
	return f.readPos, nil
}

func (f *WebDavFile) Stat() (os.FileInfo, error) {
	return &f.nameSnapshot, nil
}

func (f *WebDavFile) Write(p []byte) (int, error) {
	count,err := f.panClientProxy.UploadFilePart(f.userId, f.nameSnapshot.fullPath, f.writePos, p)
	if err != nil {
		return 0, err
	}
	f.writePos += int64(count)
	return count, nil
}






// WebDavFileInfo 文件信息
type WebDavFileInfo struct {
	os.FileInfo
	fileId string
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	fullPath string
}

func NewWebDavFileInfo(fileItem *aliyunpan.FileEntity) WebDavFileInfo  {
	var LOC, _ = time.LoadLocation("Asia/Shanghai")
	t,_ := time.ParseInLocation("2006-01-02 15:04:05", fileItem.UpdatedAt, LOC)
	fm := os.FileMode(0)
	if fileItem.IsFolder() {
		fm = os.ModeDir
	}
	return WebDavFileInfo{
		fileId: fileItem.FileId,
		name:    fileItem.FileName,
		size:    fileItem.FileSize,
		mode:    fm,
		modTime: t,
		fullPath: fileItem.Path,
	}
}

func (f *WebDavFileInfo) Name() string       { return f.name }
func (f *WebDavFileInfo) Size() int64        { return f.size }
func (f *WebDavFileInfo) Mode() os.FileMode  { return f.mode }
func (f *WebDavFileInfo) ModTime() time.Time { return f.modTime }
func (f *WebDavFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f *WebDavFileInfo) Sys() interface{}   { return nil }
func (f *WebDavFileInfo) ContentType(ctx context.Context) (contentType string, err error) {
	if mimeType := mime.TypeByExtension(path.Ext(f.Name())); mimeType != "" {
		// We can figure out the mime from the extension.
		return mimeType, nil
	} else {
		// We can't figure out the mime type without sniffing, call it an octet stream.
		return "application/octet-stream", nil
	}
}
