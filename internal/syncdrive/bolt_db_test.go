package syncdrive

import (
	"context"
	"fmt"
	"github.com/tickstep/bolt"
	"strings"
	"testing"
	"time"
)

func TestPath(t *testing.T) {
	db, err := bolt.Open("D:\\smb\\feny\\goprojects\\dev\\sync_drive\\5b2d7c10-e927-4e72-8f9d-5abb3bb04814\\test.db", 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return
	}
	defer db.Close()

	// add item
	// Start a writable transaction.
	tx, err := db.Begin(true)
	if err != nil {
		return
	}
	defer tx.Rollback()

	rootBucket, _ := tx.CreateBucketIfNotExists([]byte("/"))
	if e := rootBucket.Put([]byte("test"), []byte("test value")); e != nil {
		return
	}
	_, e := rootBucket.CreateBucketIfNotExists([]byte("test"))
	println(e)

	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		return
	}
}

func dosomething(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("playing")
			return
		default:
			fmt.Println("I am working!")
			time.Sleep(time.Second)
		}
	}
}

func TestContext(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Second)
		cancelFunc()
	}()
	dosomething(ctx)
}

func TestBolt(t *testing.T) {
	localFileDb := NewLocalSyncDb("D:\\smb\\feny\\goprojects\\dev\\sync_drive\\local.db")
	localFileDb.Open()
	defer localFileDb.Close()
	localFileDb.Add(&LocalFileItem{
		FileName:      "dev",
		FileSize:      0,
		FileType:      "folder",
		CreatedAt:     "2022-05-12 10:21:14",
		UpdatedAt:     "2022-05-12 10:21:14",
		FileExtension: ".db",
		Sha1Hash:      "",
		Path:          "D:\\smb\\feny\\goprojects\\dev",
	})
	localFileDb.Add(&LocalFileItem{
		FileName:      "file1.db",
		FileSize:      0,
		FileType:      "file",
		CreatedAt:     "2022-05-12 10:21:14",
		UpdatedAt:     "2022-05-12 10:21:14",
		FileExtension: ".db",
		Sha1Hash:      "",
		Path:          "D:\\smb\\feny\\goprojects\\dev\\file1.db",
	})
	go func(db LocalSyncDb) {
		for i := 1; i <= 10; i++ {
			sb := &strings.Builder{}
			fmt.Fprintf(sb, "D:\\smb\\feny\\goprojects\\dev\\go\\file%d.db", i)
			db.Add(&LocalFileItem{
				FileName:      "file1.db",
				FileSize:      0,
				FileType:      "file",
				CreatedAt:     "2022-05-12 10:21:14",
				UpdatedAt:     "2022-05-12 10:21:14",
				FileExtension: ".db",
				Sha1Hash:      "",
				Path:          sb.String(),
			})
		}
	}(localFileDb)
	time.Sleep(1 * time.Second)
	localFileDb.Add(&LocalFileItem{
		FileName:      "file1.db",
		FileSize:      0,
		FileType:      "file",
		CreatedAt:     "2022-05-12 10:21:14",
		UpdatedAt:     "2022-05-12 10:21:14",
		FileExtension: ".db",
		Sha1Hash:      "",
		Path:          "D:\\smb\\feny\\goprojects\\dev\\file3.db",
	})
	time.Sleep(5 * time.Second)
}
