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
package filter

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"

	"github.com/DomZippilli/gcs-proxy-cloud-function/main/common"
)

// ZipFilter compresses the media to a zip file with a single file inside.
//
// This is an example of a streaming filter. This will use very little memory
// and add very little latency to responses.
func ZipFilter(ctx context.Context, handle MediaFilterHandle) error {
	defer handle.input.Close()
	defer handle.output.Close()
	// delete content-length header. It is no longer accurate.
	handle.response.Header().Del("Content-Length")
	// make the archive and a single file inside it, using response as the output
	zw := zip.NewWriter(handle.output)
	defer zw.Close()
	// stream the input into the single file in the output archive
	objectName := common.ConvertURLtoObject(handle.request.URL.String())
	zipfile, err := zw.Create(path.Base(objectName))
	if err != nil {
		return fmt.Errorf("zip filter: %v", err)
	}
	io.Copy(zipfile, handle.input)
	return nil
}

// UnzipFilter will unzip an object and send the first file found, uncompressed.
//
// This is an example of a store-and-forward filter, in that it loads the
// entire response to perform its transformation, so it will use memory at least
// equal to the source, and add its processing time to latency.
func UnzipFilter(ctx context.Context, handle MediaFilterHandle) error {
	defer handle.input.Close()
	defer handle.output.Close()
	// delete content-length header. It is no longer accurate.
	handle.response.Header().Del("Content-Length")
	// read the contents into memory, since we need io.ReadAt :(
	media := new(bytes.Buffer)
	io.Copy(media, handle.input)
	// read the first file and send it uncompressed
	mediaBytes := media.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(mediaBytes), int64(len(mediaBytes)))
	if err != nil {
		return fmt.Errorf("unzip filter: %v", err)
	}
	f := zr.File[0]
	rc, err := f.Open()
	defer rc.Close()
	_, err = io.Copy(handle.output, rc)
	if err != nil {
		return fmt.Errorf("unzip filter: %v", err)
	}
	return nil
}
