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

type SyncDb interface {
	//读取记录,返回值不会是nil
	Get(key string) (ufm *UploadedFileMeta)
	//删除单条记录
	Del(key string) error
	//根据前辍删除数据库记录，比如删除一个目录时可以连同子目录一起删除
	DelWithPrefix(prefix string) error
	Put(key string, value *UploadedFileMeta) error
	Close() error
	//读取数据库指定路径前辍的第一条记录（也作为循环获取的初始化，配置Next函数使用)
	First(prefix string) (*UploadedFileMeta, error)
	//获取指定路径前辍的的下一条记录
	Next(prefix string) (*UploadedFileMeta, error)
	//是否进行自动数据库清理
	//注: 清理规则，所有以 prefix 前辍开头并且未更新的记录都将被清理，只有在必要的时候才开启这个功能。
	AutoClean(prefix string, cleanFlag bool)
}

type autoCleanInfo struct {
	PreFix   string
	SyncTime int64
}

func OpenSyncDb(file string, bucket string) (SyncDb, error) {
	return openBoltDb(file, bucket)
}

type dbTableField struct {
	Path string
	Data []byte
}
