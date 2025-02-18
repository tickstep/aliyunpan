package global

type (
	FileSourceType string
)

const (
	// FileSource 文件，包括：资源盘文件、备份盘文件
	FileSource FileSourceType = "file"
	// AlbumSource 相册
	AlbumSource FileSourceType = "album"
)
