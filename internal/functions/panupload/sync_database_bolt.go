// Copyright (c) 2020 tickstep & chenall
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package panupload

import (
	"bytes"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/tickstep/bolt"
	"github.com/tickstep/library-go/logger"
	"time"
)

type boltDB struct {
	db        *bolt.DB
	bucket    string
	next      map[string]*boltDBScan
	cleanInfo *autoCleanInfo
}

type boltDBScan struct {
	entries []*boltKV
	off     int
	size    int
}

type boltKV struct {
	k []byte
	v []byte
}

func openBoltDb(file string, bucket string) (SyncDb, error) {
	db, err := bolt.Open(file + "_bolt.db", 0600, &bolt.Options{Timeout: 5 * time.Second})

	if err != nil {
		return nil, err
	}
	logger.Verboseln("open boltDB ok")
	return &boltDB{db: db, bucket: bucket, next: make(map[string]*boltDBScan)}, nil
}

func (db *boltDB) Get(key string) (data *UploadedFileMeta) {
	data = &UploadedFileMeta{Path: key}
	db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.bucket))
		if b == nil {
			return nil
		}
		v := b.Get([]byte(key))
		return jsoniter.Unmarshal(v, data)
	})

	return data
}

func (db *boltDB) Del(key string) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.bucket))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(key))
	})
}

func (db *boltDB) AutoClean(prefix string, cleanFlag bool) {
	if !cleanFlag {
		db.cleanInfo = nil
	} else if db.cleanInfo == nil {
		db.cleanInfo = &autoCleanInfo{
			PreFix:   prefix,
			SyncTime: time.Now().Unix(),
		}
	}
}

func (db *boltDB) clean() (count uint) {
	for ufm, err := db.First(db.cleanInfo.PreFix); err == nil; ufm, err = db.Next(db.cleanInfo.PreFix) {
		if ufm.LastSyncTime != db.cleanInfo.SyncTime {
			db.DelWithPrefix(ufm.Path)
		}
	}
	return
}

func (db *boltDB) DelWithPrefix(prefix string) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.bucket))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, _ := c.Seek([]byte(prefix)); k != nil && bytes.HasPrefix(k, []byte(prefix)); k, _ = c.Next() {
			b.Delete(k)
		}
		return nil
	})
}

func (db *boltDB) First(prefix string) (*UploadedFileMeta, error) {
	db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(db.bucket))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		db.next[prefix] = &boltDBScan{
			entries: []*boltKV{},
			off:     0,
			size:    0,
		}
		for k, v := c.Seek([]byte(prefix)); k != nil && bytes.HasPrefix(k, []byte(prefix)); k, v = c.Next() {
			//fmt.Printf("key=%s, value=%s\n", k, v)
			if len(k) > 0 {
				db.next[prefix].entries = append(db.next[prefix].entries, &boltKV{
					k: k,
					v: v,
				})
			}
		}
		db.next[prefix].off = 0
		db.next[prefix].size = len(db.next[prefix].entries)
		return nil
	})
	return db.Next(prefix)
}

func (db *boltDB) Next(prefix string) (*UploadedFileMeta, error) {
	data := &UploadedFileMeta{}
	if _,ok := db.next[prefix]; ok {
		if db.next[prefix].off >= db.next[prefix].size {
			return nil, fmt.Errorf("no any more record")
		}
		kv := db.next[prefix].entries[db.next[prefix].off]
		db.next[prefix].off++
		if kv != nil {
			jsoniter.Unmarshal(kv.v, &data)
			data.Path = string(kv.k)
			return data, nil
		}
	}
	return nil, fmt.Errorf("no any more record")
}

func (db *boltDB) Put(key string, value *UploadedFileMeta) error {
	if db.cleanInfo != nil {
		value.LastSyncTime = db.cleanInfo.SyncTime
	}

	return db.db.Update(func(tx *bolt.Tx) error {
		data, err := jsoniter.Marshal(value)
		if err != nil {
			return err
		}
		b := tx.Bucket([]byte(db.bucket))
		if b == nil {
			b,err = tx.CreateBucket([]byte(db.bucket))
			if err != nil {
				return err
			}
		}
		return b.Put([]byte(key), data)
	})
}

func (db *boltDB) Close() error {
	if db.cleanInfo != nil {
		db.clean()
	}
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}