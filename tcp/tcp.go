package tcp

import (
	"errors"
	"faultline/config"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"
)

// Local RNG for randomized faults (drop/reset probabilities), avoids deprecated global seeding.
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// Proxy represents a single TCP proxy instance with configured faults.
type Proxy struct {
	rule config.TCPRule
}

// NewProxy creates a new TCP proxy for the given rule.
func NewProxy(rule config.TCPRule) *Proxy {
	return &Proxy{rule: rule}
}

// Start begins listening on the rule.Listen address and proxies to rule.Upstream.
func (p *Proxy) Start(stop <-chan struct{}) error {
	ln, err := net.Listen("tcp", p.rule.Listen)
	if err != nil {
		return err
	}
	log.Printf("[DB] Listening on %s -> %s", p.rule.Listen, p.rule.Upstream)

	var wg sync.WaitGroup

	go func() {
		<-stop
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// Listener closed during shutdown: exit accept loop
			if errors.Is(err, net.ErrClosed) {
				break
			}
			// Temporary/timeout error: brief backoff and continue
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			// Other errors: log and backoff to avoid busy loop
			log.Printf("[DB] Accept error on %s: %v", p.rule.Listen, err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			p.handleConn(c)
		}(conn)
	}

	wg.Wait()
	return nil
}

func (p *Proxy) handleConn(client net.Conn) {
	faults := p.rule.Faults

	if faults.RefuseConnections {
		// Immediately close connection to simulate refusal
		_ = client.Close()
		return
	}

	// Optional initial latency per-connection
	if faults.LatencyMs > 0 {
		time.Sleep(time.Duration(faults.LatencyMs) * time.Millisecond)
	}

	// Randomly reset after accept
	if faults.ResetProbability > 0 && rng.Float64() < faults.ResetProbability {
		_ = client.Close()
		return
	}

	upstream, err := net.DialTimeout("tcp", p.rule.Upstream, 5*time.Second)
	if err != nil {
		log.Printf("[DB] Upstream dial error for %s: %v", p.rule.Upstream, err)
		_ = client.Close()
		return
	}

	// Bi-directional piping with optional throttling/drops
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		copyWithFaults(upstream, client, faults)
	}()

	go func() {
		defer wg.Done()
		copyWithFaults(client, upstream, faults)
	}()

	wg.Wait()
	_ = client.Close()
	_ = upstream.Close()
}

// copyWithFaults copies data from src to dst applying drop and bandwidth throttling.
func copyWithFaults(dst net.Conn, src net.Conn, f config.TCPFaults) {
	// Simple chunked copy
	bufSize := 32 * 1024
	buf := make([]byte, bufSize)
	var bwPerSec int64
	if f.BandwidthKbps > 0 {
		bwPerSec = int64(f.BandwidthKbps) * 1024 // bytes per second
	}
	var sentThisWindow int64
	windowStart := time.Now()

	for {
		// Apply per-chunk latency if configured (approximate)
		if f.LatencyMs > 0 {
			time.Sleep(time.Duration(f.LatencyMs) * time.Millisecond)
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			// Randomly drop this chunk
			if f.DropProbability > 0 && rng.Float64() < f.DropProbability {
				// drop silently
			} else {
				// Bandwidth throttling: ensure we don't exceed bwPerSec
				if bwPerSec > 0 {
					now := time.Now()
					if now.Sub(windowStart) >= time.Second {
						windowStart = now
						sentThisWindow = 0
					}
					// If sending this chunk would exceed budget, sleep
					if sentThisWindow+int64(n) > bwPerSec {
						sleepDur := time.Second - now.Sub(windowStart)
						if sleepDur > 0 {
							time.Sleep(sleepDur)
							windowStart = time.Now()
							sentThisWindow = 0
						}
					}
				}

				wn, writeErr := dst.Write(buf[:n])
				sentThisWindow += int64(wn)
				if writeErr != nil {
					return
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return
			}
			return
		}
	}
}
