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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/DomZippilli/gcs-proxy-cloud-function/common"
	"github.com/DomZippilli/gcs-proxy-cloud-function/filter"

	storage "cloud.google.com/go/storage"
	"github.com/rs/zerolog/log"
)

// Read returns objects from a GCS bucket, mapping the URL to object names.
// Media caching is bypassed.
func Read(ctx context.Context, response http.ResponseWriter,
	request *http.Request, pipeline filter.Pipeline) {
	noCache := func(s string) ([]byte, bool) {
		return nil, false
	}
	ReadWithCache(ctx, response, request, pipeline, noCache, filter.Pipeline{})
}

// CacheGet defines how CachedGet will try to get media from the cache.
type CacheGet func(string) ([]byte, bool)

// ReadWithCache returns objects from a GCS bucket, mapping the URL to object names.
// Cached media may be served, sparing a trip to GCS.
//
// Filters in missPipeline will be applied on cache misses. A cache fill
// filter is a good idea here.
//
// Filters in hitPipeline will be applied on cache hits. Reducing the pipeline
// to not repeat steps done on fill (e.g., compression, transcoding) is a good
// idea here.
func ReadWithCache(ctx context.Context, response http.ResponseWriter,
	request *http.Request, missPipeline filter.Pipeline, cacheGet CacheGet,
	hitPipeline filter.Pipeline) {
	// normalize path
	objectName := common.NormalizePath(request.URL.Path)

	// get the object handle and headers. Headers are always cached and obey
	// Cache-Control header, so this will not call GCS unless there's a miss.
	// In general, header hits and media hits should line up.
	objectHandle := gcs.Bucket(bucket).Object(objectName)
	// get static-serving metadata and set headers
	err := setHeaders(ctx, objectHandle, response)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			http.Error(response, "", http.StatusNotFound)
			return
		} else {
			log.Error().Msgf("get: %v", err)
		}
	}

	// try the media cache
	var media io.Reader
	var pipeline filter.Pipeline
	maybeMedia, hit := cacheGet(objectName)
	if hit {
		log.Debug().Msgf("gcs getwithcache: HIT")
		media = bytes.NewReader(maybeMedia)
		// transformations may be cached; use cached content length
		response.Header().Set("Content-Length", fmt.Sprint(len(maybeMedia)))
		pipeline = hitPipeline
	} else {
		log.Debug().Msgf("gcs getwithcache: MISS")
		// get object content and send it
		// TODO(domz): need an aggressive reader
		objectContent, err := objectHandle.NewReader(ctx)
		if err != nil {
			log.Error().Msgf("get: %v", err)
		}
		defer objectContent.Close()
		media = objectContent
		pipeline = missPipeline
	}

	// serve the media
	if len(pipeline) > 0 {
		// use a filter pipeline
		_, err = filter.PipelineCopy(ctx, response, media, request, pipeline)
	} else {
		// unfiltered, simple copy
		_, err = io.Copy(response, media)
	}
	if err != nil {
		log.Error().Msgf("get: %v", err)
	}
	return
}
