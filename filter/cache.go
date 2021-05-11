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
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/DomZippilli/gcs-proxy-cloud-function/common"
	"github.com/rs/zerolog/log"
)

type CacheSet func(string, []byte, time.Duration)

// FillCache will tee the media it recieves into a cache, using the normalized
// request URL as the key. Supply a cache setter with the setter argument.
func FillCache(ctx context.Context, handle MediaFilterHandle, setter CacheSet) error {
	defer handle.input.Close()
	defer handle.output.Close()
	// create a buffer for the media
	cachedMedia := new(bytes.Buffer)
	// create a tee from the input that writes to cachedMedia
	tee := io.TeeReader(handle.input, cachedMedia)
	// write the response through the tee
	if _, err := io.Copy(handle.output, tee); err != nil {
		return fmt.Errorf("fillcache: %v", err)
	}
	// determine expiration
	cacheExpiration := 0 * time.Second
	cacheControl := handle.response.Header().Get("Cache-Control")
	if cacheControl != "" &&
		strings.HasPrefix(cacheControl, "max-age") {
		ccSecs, err := strconv.Atoi(strings.Split(cacheControl, "=")[1])
		if err != nil {
			log.Error().Msgf("fillcache: %v", err)
		} else {
			cacheExpiration = time.Second * time.Duration(ccSecs)
		}
	}
	// cache the media
	cacheKey := common.NormalizeURL(handle.request.URL.String())
	setter(cacheKey, cachedMedia.Bytes(), cacheExpiration)
	return nil
}
