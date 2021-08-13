// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package common

import (
	"fmt"
	"io"
	"net/http"
    "strings"
)

// NormalizePath:
//   replace trailing slashes with "/index.html";
//   remove leading slashes.
func NormalizePath(path string) (object string) {
    if strings.HasSuffix(path, "/") {
        path = path + "index.html"
    }
    return strings.TrimLeft(path, "/")
}

func GetRuntimeProjectId() (string, error) {
	// Define the metadata request.
	client := &http.Client{}
	req, err := http.NewRequest("GET",
		"http://metadata.google.internal/computeMetadata/v1/project/project-id",
		nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Metadata-Flavor", "Google")
	// Make the request.
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// Read the response and return it.
	bodyBytes, err := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("get project id: %v", string(bodyBytes))
	}
	return string(bodyBytes), nil
}
