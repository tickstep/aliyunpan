// Copyright (c) 2020 tickstep.
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
package panupdate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func update(targetPath string, src io.Reader) error {
	info, err := os.Stat(targetPath)
	if err != nil {
		fmt.Printf("Warning: %s\n", err)
		return nil
	}

	privMode := info.Mode()

	oldPath := filepath.Join(filepath.Dir(targetPath), "old-"+filepath.Base(targetPath))

	err = os.Rename(targetPath, oldPath)
	if err != nil {
		return err
	}

	newFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, privMode)
	if err != nil {
		return err
	}

	_, err = io.Copy(newFile, src)
	if err != nil {
		return err
	}

	err = newFile.Close()
	if err != nil {
		fmt.Printf("Warning: 关闭文件发生错误: %s\n", err)
	}

	err = os.Remove(oldPath)
	if err != nil {
		fmt.Printf("Warning: 移除旧文件发生错误: %s\n", err)
	}
	return nil
}
