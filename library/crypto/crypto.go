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
package crypto

import (
	"fmt"
	"github.com/tickstep/library-go/archive"
	"github.com/tickstep/library-go/crypto"
	"io"
	"os"
	"strings"
)

// CryptoMethodSupport 检测是否支持加密解密方法
func CryptoMethodSupport(method string) bool {
	switch method {
	case "aes-128-ctr", "aes-192-ctr", "aes-256-ctr", "aes-128-cfb", "aes-192-cfb", "aes-256-cfb", "aes-128-ofb", "aes-192-ofb", "aes-256-ofb":
		return true
	}

	return false
}

// EncryptFile 加密本地文件
func EncryptFile(method string, key []byte, filePath string, isGzip bool) (encryptedFilePath string, err error) {
	if !CryptoMethodSupport(method) {
		return "", fmt.Errorf("unknown encrypt method: %s", method)
	}

	if isGzip {
		err = archive.GZIPCompressFile(filePath)
		if err != nil {
			return
		}
	}

	plainFile, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return
	}

	defer plainFile.Close()

	var cipherReader io.Reader
	switch method {
	case "aes-128-ctr":
		cipherReader, err = crypto.Aes128CTREncrypt(crypto.Convert16bytes(key), plainFile)
	case "aes-192-ctr":
		cipherReader, err = crypto.Aes192CTREncrypt(crypto.Convert24bytes(key), plainFile)
	case "aes-256-ctr":
		cipherReader, err = crypto.Aes256CTREncrypt(crypto.Convert32bytes(key), plainFile)
	case "aes-128-cfb":
		cipherReader, err = crypto.Aes128CFBEncrypt(crypto.Convert16bytes(key), plainFile)
	case "aes-192-cfb":
		cipherReader, err = crypto.Aes192CFBEncrypt(crypto.Convert24bytes(key), plainFile)
	case "aes-256-cfb":
		cipherReader, err = crypto.Aes256CFBEncrypt(crypto.Convert32bytes(key), plainFile)
	case "aes-128-ofb":
		cipherReader, err = crypto.Aes128OFBEncrypt(crypto.Convert16bytes(key), plainFile)
	case "aes-192-ofb":
		cipherReader, err = crypto.Aes192OFBEncrypt(crypto.Convert24bytes(key), plainFile)
	case "aes-256-ofb":
		cipherReader, err = crypto.Aes256OFBEncrypt(crypto.Convert32bytes(key), plainFile)
	default:
		return "", fmt.Errorf("unknown encrypt method: %s", method)
	}

	if err != nil {
		return
	}

	plainFileInfo, err := plainFile.Stat()
	if err != nil {
		return
	}

	encryptedFilePath = filePath + ".encrypt"
	encryptedFile, err := os.OpenFile(encryptedFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, plainFileInfo.Mode())
	if err != nil {
		return
	}

	defer encryptedFile.Close()

	_, err = io.Copy(encryptedFile, cipherReader)
	if err != nil {
		return
	}

	os.Remove(filePath)

	return encryptedFilePath, nil
}

// DecryptFile 加密本地文件
func DecryptFile(method string, key []byte, filePath string, isGzip bool) (decryptedFilePath string, err error) {
	if !CryptoMethodSupport(method) {
		return "", fmt.Errorf("unknown decrypt method: %s", method)
	}

	cipherFile, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return
	}

	var plainReader io.Reader
	switch method {
	case "aes-128-ctr":
		plainReader, err = crypto.Aes128CTRDecrypt(crypto.Convert16bytes(key), cipherFile)
	case "aes-192-ctr":
		plainReader, err = crypto.Aes192CTRDecrypt(crypto.Convert24bytes(key), cipherFile)
	case "aes-256-ctr":
		plainReader, err = crypto.Aes256CTRDecrypt(crypto.Convert32bytes(key), cipherFile)
	case "aes-128-cfb":
		plainReader, err = crypto.Aes128CFBDecrypt(crypto.Convert16bytes(key), cipherFile)
	case "aes-192-cfb":
		plainReader, err = crypto.Aes192CFBDecrypt(crypto.Convert24bytes(key), cipherFile)
	case "aes-256-cfb":
		plainReader, err = crypto.Aes256CFBDecrypt(crypto.Convert32bytes(key), cipherFile)
	case "aes-128-ofb":
		plainReader, err = crypto.Aes128OFBDecrypt(crypto.Convert16bytes(key), cipherFile)
	case "aes-192-ofb":
		plainReader, err = crypto.Aes192OFBDecrypt(crypto.Convert24bytes(key), cipherFile)
	case "aes-256-ofb":
		plainReader, err = crypto.Aes256OFBDecrypt(crypto.Convert32bytes(key), cipherFile)
	default:
		return "", fmt.Errorf("unknown decrypt method: %s", method)
	}

	if err != nil {
		return
	}

	cipherFileInfo, err := cipherFile.Stat()
	if err != nil {
		return
	}

	decryptedFilePath = strings.TrimSuffix(filePath, ".encrypt")
	decryptedTmpFilePath := decryptedFilePath + ".decrypted"
	decryptedTmpFile, err := os.OpenFile(decryptedTmpFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, cipherFileInfo.Mode())
	if err != nil {
		return
	}

	_, err = io.Copy(decryptedTmpFile, plainReader)
	if err != nil {
		return
	}

	decryptedTmpFile.Close()
	cipherFile.Close()

	if isGzip {
		err = archive.GZIPUnompressFile(decryptedTmpFilePath)
		if err != nil {
			os.Remove(decryptedTmpFilePath)
			return
		}

		// 删除已加密的文件
		os.Remove(filePath)
	}

	if filePath != decryptedFilePath {
		os.Rename(decryptedTmpFilePath, decryptedFilePath)
	} else {
		decryptedFilePath = decryptedTmpFilePath
	}

	return decryptedFilePath, nil
}
