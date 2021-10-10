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

type (
	// AssetInfo asset 信息
	AssetInfo struct {
		Name               string `json:"name"`
		ContentType        string `json:"content_type"`
		State              string `json:"state"`
		Size               int64  `json:"size"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}

	// ReleaseInfo 发布信息
	ReleaseInfo struct {
		TagName string       `json:"tag_name"`
		Assets  []*AssetInfo `json:"assets"`
	}
)
