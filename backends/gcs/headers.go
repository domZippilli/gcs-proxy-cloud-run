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
package gcs

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	storage "cloud.google.com/go/storage"
	cache "github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
)

// objectMetadataCache stores object metadata to speed up serving of data.
// The data itself is not cached, just values like Content-Type, Cache-Control,
// etc.
// TODO(domz): Small data, but still, need memory bounds.
var objectMetadataCache = cache.New(90*time.Second, 10*time.Minute)

// setHeaders will transfer HTTP headers from GCS metadata to the response.
func setHeaders(ctx context.Context, objectHandle *storage.ObjectHandle,
	response http.ResponseWriter) (err error) {

	// get object metadata. Use a cache to speed up TTFB.
	objectAttrs, err := getAttrs(ctx, objectHandle)
	if err != nil {
		return err
	}

	// set all headers
	if objectAttrs.CacheControl != "" {
		response.Header().Set("Cache-Control", objectAttrs.CacheControl)
	}
	if objectAttrs.ContentEncoding != "" {
		response.Header().Set("Content-Encoding", objectAttrs.ContentEncoding)
	}
	if objectAttrs.ContentLanguage != "" {
		response.Header().Set("Content-Language", objectAttrs.ContentLanguage)
	}
	if objectAttrs.ContentType != "" {
		response.Header().Set("Content-Type", objectAttrs.ContentType)
	}
	response.Header().Set("Content-Length", fmt.Sprint(objectAttrs.Size))
	return
}

// getAttrs will get the metadata of an object, using a local cache to
// store metadata and avoid repeated metadata GETs.
func getAttrs(ctx context.Context, objectHandle *storage.ObjectHandle) (
	objectAttrs *storage.ObjectAttrs, err error) {
	// get object metadata. Use a cache to speed up TTFB.
	maybeAttrs, hit := objectMetadataCache.Get(objectHandle.ObjectName())
	if hit {
		objectAttrs = maybeAttrs.(*storage.ObjectAttrs)
	} else {
		// TODO(domz): no need for full projection here
		objectAttrs, err = objectHandle.Attrs(ctx)
		if err != nil {
			return
		}
		// cache the result, honoring Cache-Control: max-age
		expiry := cache.DefaultExpiration
		cacheControl := objectAttrs.CacheControl
		if cacheControl != "" &&
			strings.HasPrefix(cacheControl, "max-age") {
			ccSecs, err := strconv.Atoi(strings.Split(cacheControl, "=")[1])
			if err != nil {
				log.Fatal().Msgf("getAttrs: %v", err)
			} else {
				expiry = time.Second * time.Duration(ccSecs)
			}
		}
		objectMetadataCache.Set(objectHandle.ObjectName(), objectAttrs, expiry)
	}
	return
}
