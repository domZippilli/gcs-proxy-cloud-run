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
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"
	"sync/atomic"
	"time"

	storage "cloud.google.com/go/storage"
	"github.com/DomZippilli/gcs-proxy-cloud-function/common"
	"github.com/rs/zerolog/log"
)

var minimum_stream_size int64 = common.AsBytes(common.MB, 8)
var numWorkers int64 = int64(runtime.NumCPU())

// managedDownload evaluates an object for whether to download it in a
// single stream or in concurrent streams, and in either case returns an
// io.Reader representing the object media.
func managedDownload(ctx context.Context, objectHandle *storage.ObjectHandle) (io.ReadCloser, error) {
	// get decision factors
	objectAttrs, err := objectHandle.Attrs(ctx)
	if err != nil {
		return nil, err
	}
	objectSize := objectAttrs.Size
	streamCount := numWorkers
	streamSize := int64(objectSize / streamCount)
	// make decision
	// less than one stream makes no sense; small streams not worth the overhead.
	shouldDoConcurrent := streamCount > 1 && streamSize > minimum_stream_size
	if shouldDoConcurrent {
		log.Debug().Msgf("Multi-stream (%v) download for %v", streamCount, objectHandle.ObjectName())
		return doConcurrentDownload(ctx, objectHandle, objectSize, streamCount)
	}
	log.Debug().Msgf("Single-stream download for %v", objectHandle.ObjectName())
	return objectHandle.NewReader(ctx)
}

func doConcurrentDownload(ctx context.Context,
	objectHandle *storage.ObjectHandle, objectSize, streamCount int64) (io.ReadCloser, error) {
	// compute ranges, make slices of pipe reader/writer to fit
	downloadRanges := subdivideRange(0, objectSize, streamCount)
	rangeReaders := []io.Reader{}
	// launch goroutines to fill each pipe
	for i, downloadRange := range downloadRanges {
		rBegin, rEnd := downloadRange.begin, downloadRange.end+1
		rSize := rEnd - rBegin
		// create the pipe
		pr, pw := io.Pipe()
		// wrap the reader and store
		rangeReaders = append(rangeReaders, io.Reader(pr))
		// buffer the writer for the goroutine
		var bpw *bufio.Writer
		if i == 0 {
			// The first slice should send bytes right away while the rest fill.
			bpw = bufio.NewWriterSize(pw, int(common.AsBytes(common.MB, 2)))
		} else {
			bpw = bufio.NewWriterSize(pw, int(rSize))
		}
		// fill the pipe concurrently
		errs := make(chan error, 1)
		go func() {
			defer close(errs)
			rr, err := newParallelRangeReader(ctx, objectHandle.BucketName(), objectHandle.ObjectName(), rBegin, rEnd)
			if err != nil {
				pw.CloseWithError(err)
				errs <- fmt.Errorf("newParallelRangeReader: %v", err)
			}
			defer rr.Close()
			defer pw.Close()
			defer bpw.Flush()
			log.Debug().Msgf("concurrent stream %v range %v %v", objectHandle.ObjectName(), rBegin, rEnd)
			if _, err = io.CopyN(bpw, rr, rSize); err != nil {
				// TODO(domz): There's always one EOF error here, probably the last one, not sure there should be.
				errs <- fmt.Errorf("CopyN: %v", err)
			}
			return
		}()
		// watch for errors
		// TODO(domz): probably the thing to do is somehow collect these
		// errors and if there is one, 500 the response. But do this without blocking
		// this function any longer than necessary.
		go func(c context.Context) {
			for {
				if err, open := <-errs; open {
					log.Error().Msgf("maybeConcurrentDownload: %v", err)
				} else {
					break
				}
				time.Sleep(time.Millisecond)
			}
		}(ctx)
	}
	// using the magic of MultiReader, concat the slice of buffers as a single stream
	// wrap in NopCloser to get an io.ReadCloser like GCS SDK's Reader
	return ioutil.NopCloser(io.MultiReader(rangeReaders...)), nil
}

var clientPool []*storage.Client
var clientPoolInitialized bool = false
var clientIdx int32 = -1

func initializeClientPool(workers int) error {
	if !clientPoolInitialized {
		clientPool = []*storage.Client{}
		for i := 0; i < workers; i++ {
			log.Debug().Msgf("initializing new client %v", i)
			gcs, err := storage.NewClient(context.Background())
			if err != nil {
				return err
			}
			clientPool = append(clientPool, gcs)
		}
	}
	return nil
}

func getClientFromPool() *storage.Client {
	atomic.AddInt32(&clientIdx, 1)
	if clientIdx > int32(len(clientPool)-1) {
		ocid := clientIdx
		atomic.CompareAndSwapInt32(&clientIdx, ocid, 0)
	}
	log.Debug().Msgf("handing out client %v", clientIdx)
	return clientPool[clientIdx]
}

// newParallelRangeReader prepares a range-specific storage.Reader with its own client, and returns a pointer to it.
func newParallelRangeReader(ctx context.Context, bucket, object string, rBegin, rEnd int64) (*storage.Reader, error) {
	gcs := getClientFromPool()
	rr, err := gcs.Bucket(bucket).Object(object).NewRangeReader(ctx, rBegin, rEnd)
	if err != nil {
		return nil, err
	}
	return rr, nil
}

// min returns the lesser of two int64 values (no this is not built into Go)
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// numberRange is just a beginning and end of a range of numbers.
type numberRange struct {
	begin int64
	end   int64
}

// subdivideRange generates n exclusive subdivisions of a number range.
func subdivideRange(rangeStart, rangeEnd, subdivisions int64) []numberRange {
	rangeSize := rangeEnd - rangeStart
	subrangeSize := int64(rangeSize / subdivisions)
	var ranges []numberRange
	start := rangeStart
	var finish int64 = -1
	for finish < rangeEnd {
		finish = start + subrangeSize
		ranges = append(ranges, numberRange{
			begin: start,
			end:   min(finish, rangeEnd),
		})
		start = finish + 1
	}
	return ranges
}
