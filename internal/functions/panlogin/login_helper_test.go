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
package panlogin

import (
	"fmt"
	"github.com/tickstep/library-go/ids"
	"testing"
	"time"
)

func TestGetQRCodeLoginUrl(t *testing.T) {
	h := NewLoginHelper("http://localhost:8977")
	keyStr := ids.GetUniqueId("", 32)
	fmt.Println(keyStr)
	r, e := h.GetQRCodeLoginUrl(keyStr)
	fmt.Println(e)
	fmt.Println(r)
}

func TestGetQRCodeLoginResult(t *testing.T) {
	h := NewLoginHelper("http://localhost:8977")
	tokenId := "26e69f9978ba4574a1d66e58399fed4e"
	for {
		r, e := h.GetQRCodeLoginResult(tokenId)
		fmt.Println(e)
		fmt.Println(r)
		time.Sleep(1 * time.Second)
		if r.QrCodeStatus == "CONFIRMED" {
			break
		}
	}
}

func TestGetRefreshToken(t *testing.T) {
	h := NewLoginHelper("http://localhost:8977")
	tokenId := "26e69f9978ba4574a1d66e58399fed4e"
	r, e := h.GetRefreshToken(tokenId)
	fmt.Println(e)
	fmt.Println(r)
}

func TestParseRefreshToken(t *testing.T) {
	h := NewLoginHelper("http://localhost:8977")
	secureToken := "eff6054736d47b9c31f8839465555ebdff38c878ea0abbcd4b2336b30d33c71c7dac8824e40991654429ae521d8ef471"
	r, e := h.ParseSecureRefreshToken("", secureToken)
	fmt.Println(e)
	fmt.Println(r)
}
