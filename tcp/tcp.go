package tcp

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	"faultline/config"
)

// Proxy is a TCP proxy with fault injection knobs.
// It listens on rule.Listen and forwards to rule.Upstream with faults.
type Proxy struct {
	rule config.TCPRule
}

func NewProxy(rule config.TCPRule) *Proxy {
	return &Proxy{rule: rule}
}

// Start begins accepting connections and proxying data until stop is closed.
func (p *Proxy) Start(stop <-chan struct{}) error {
	if p.rule.Faults.RefuseConnections {
		log.Printf("[DB] Refusing connections on %s (configured)", p.rule.Listen)
	}

	ln, err := net.Listen("tcp", p.rule.Listen)
	if err != nil {
		return fmt.Errorf("listen %s: %w", p.rule.Listen, err)
	}
	defer ln.Close()
	log.Printf("[DB] Listening on %s -> %s", p.rule.Listen, p.rule.Upstream)

	var tempDelay time.Duration
	for {
		ln.(*net.TCPListener).SetDeadline(time.Now().Add(500 * time.Millisecond))
		conn, err := ln.Accept()
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			select {
			case <-stop:
				return nil
			default:
				continue
			}
		}
		if err != nil {
			if tempDelay == 0 {
				tempDelay = 5 * time.Millisecond
			} else {
				tempDelay *= 2
			}
			if max := 1 * time.Second; tempDelay > max {
				tempDelay = max
			}
			log.Printf("[DB] accept error: %v; retrying in %v", err, tempDelay)
			time.Sleep(tempDelay)
			continue
		}
		tempDelay = 0

		go p.handleConn(conn)
	}
}

func (p *Proxy) handleConn(client net.Conn) {
	defer client.Close()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	f := p.rule.Faults

	if f.RefuseConnections {
		// Simulate immediate refusal
		return
	}

	if f.LatencyMs > 0 {
		time.Sleep(time.Duration(f.LatencyMs) * time.Millisecond)
	}

	server, err := net.DialTimeout("tcp", p.rule.Upstream, 5*time.Second)
	if err != nil {
		log.Printf("[DB] dial upstream %s failed: %v", p.rule.Upstream, err)
		return
	}
	defer server.Close()

	if rng.Float64() < f.ResetProbability {
		// Close abruptly to simulate connection reset after accept
		return
	}

	var clientToServerBytes, serverToClientBytes uint64
	done := make(chan struct{}, 2)

	go func() {
		atomic.AddUint64(&clientToServerBytes, uint64(copyWithFaults(server, client, f, rng)))
		done <- struct{}{}
	}()
	go func() {
		atomic.AddUint64(&serverToClientBytes, uint64(copyWithFaults(client, server, f, rng)))
		done <- struct{}{}
	}()

	<-done
	<-done
	log.Printf("[DB] %s <-> %s closed. c->s=%dB s->c=%dB", client.RemoteAddr(), p.rule.Upstream, clientToServerBytes, serverToClientBytes)
}

func copyWithFaults(dst io.Writer, src io.Reader, f config.TCPFaults, rng *rand.Rand) int {
	buf := make([]byte, 32*1024)
	total := 0
	var lastSleep time.Time

	for {
		n, readErr := src.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			// Random drop
			if f.DropProbability > 0 && rng.Float64() < f.DropProbability {
				continue
			}
			// Bandwidth throttle
			if f.BandwidthKbps > 0 {
				perChunkDelay := time.Duration(float64(n) / (float64(f.BandwidthKbps) * 1024.0) * float64(time.Second))
				if perChunkDelay > 0 {
					// Avoid tight loop sleeps
					if time.Since(lastSleep) < perChunkDelay {
						time.Sleep(perChunkDelay - time.Since(lastSleep))
					}
					lastSleep = time.Now()
				}
			}
			written, writeErr := dst.Write(chunk)
			total += written
			if writeErr != nil {
				return total
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return total
			}
			return total
		}
	}
}
