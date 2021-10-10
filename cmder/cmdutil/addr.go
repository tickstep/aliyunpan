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
package cmdutil

import (
	"net"
)

// ListAddresses 列出本地可用的 IP 地址
func ListAddresses() (addresses []string) {
	iFaces, _ := net.Interfaces()
	addresses = make([]string, 0, len(iFaces))
	for k := range iFaces {
		iFaceAddrs, _ := iFaces[k].Addrs()
		for l := range iFaceAddrs {
			switch v := iFaceAddrs[l].(type) {
			case *net.IPNet:
				addresses = append(addresses, v.IP.String())
			case *net.IPAddr:
				addresses = append(addresses, v.IP.String())
			}
		}
	}
	return
}

// ParseHost 解析地址中的host
func ParseHost(address string) string {
	h, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	return h
}
