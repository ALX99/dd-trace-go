// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

//go:build !go1.22

// TODO(knusbaum): This file should be deleted once go1.21 falls out of support
package tracer

import (
	cryptorand "crypto/rand"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
)

var (
	random   randT
	warnOnce sync.Once
	seedSeq  int64
	randPool = sync.Pool{
		New: func() interface{} {
			var seed int64
			n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(math.MaxInt64))
			if err == nil {
				seed = n.Int64()
			} else {
				warnOnce.Do(func() {
					log.Warn("cannot generate random seed: %v; using current time", err)
				})
				seed = time.Now().UnixNano()
			}
			// seedSeq makes sure we don't create two generators with the same seed
			// by accident.
			return rand.New(rand.NewSource(seed + atomic.AddInt64(&seedSeq, 1)))
		},
	}
)

type randT struct{}

// Uint64 returns a random number. It's optimized for concurrent access.
func (randT) Uint64() uint64 {
	// sync.Pool is optimized so we end up with one *rand.Rand per P under load.
	// This is pretty much optimal for avoiding contention.
	r := randPool.Get().(*rand.Rand)
	// NOTE: TestTextMapPropagator fails if we return r.Uint64() here. Seems like
	// span ids are expected to be 64 bit with the first bit being 0?
	v := uint64(r.Int63())
	randPool.Put(r)
	return v
}

// generateSpanID returns a random uint64 that has been XORd with the startTime.
// This is done to get around the 32-bit random seed limitation that may create collisions if there is a large number
// of go services all generating spans.
func generateSpanID(startTime int64) uint64 {
	return random.Uint64() ^ uint64(startTime)
}

func randUint64() uint64 {
	return random.Uint64()
}
