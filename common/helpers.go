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
)

// NormalizeURL removes the leading slash from URLs, and also
// redirects root requests "/" to index.html.
func NormalizeURL(url string) (object string) {
	switch url {
	case "/":
		return "index.html"
	default:
		return url[1:]
	}
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

type ByteCount int

const (
	KB ByteCount = iota
	MB ByteCount = iota
	GB ByteCount = iota
	TB ByteCount = iota
	PB ByteCount = iota
)

// Megabytes takes a number of megabytes, and returns it in bytes.
func AsBytes(unit ByteCount, quantity int64) int64 {
	switch unit {
	case PB:
		return quantity * 1024 * 1024 * 1024 * 1024 * 1024
	case TB:
		return quantity * 1024 * 1024 * 1024 * 1024
	case GB:
		return quantity * 1024 * 1024 * 1024
	case MB:
		return quantity * 1024 * 1024
	default:
		// KB
		return quantity * 1024
	}
}
