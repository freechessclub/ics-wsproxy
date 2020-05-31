// Copyright © 2020 Free Chess Club <help@freechess.club>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ziutek/telnet"
)

const (
	// dial timeout
	timeout = 5 * time.Second
	// keep alive timeout
	keepAlive = 80 * time.Second
)

// Proxy is a ICS to WebSocket Proxy
type Proxy struct {
	ws               *websocket.Conn
	ics              *telnet.Conn
	keepAliveChannel chan struct{}
	icsReaderChannel chan struct{}
	wsReaderChannel  chan struct{}
	shutdown         bool
	sync.Mutex
}

// NewProxy creates a new ICS to WebSocket proxy
func NewProxy(addr string, ws *websocket.Conn) (*Proxy, error) {
	ics, err := telnet.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}

	proxy := &Proxy{
		ws:               ws,
		ics:              ics,
		keepAliveChannel: make(chan struct{}),
		icsReaderChannel: make(chan struct{}),
		wsReaderChannel:  make(chan struct{}),
		shutdown:         false,
	}

	go proxy.keepAlive()
	go proxy.icsReader()
	go proxy.wsReader()

	return proxy, nil
}

// Shutdown shuts down the proxy
func (p *Proxy) Shutdown() {
	p.Lock()
	if !p.shutdown {
		p.ws.WriteMessage(websocket.CloseMessage, []byte{})
		p.ws.Close()
		close(p.wsReaderChannel)
		close(p.icsReaderChannel)
		close(p.keepAliveChannel)
		p.shutdown = true
	}
	p.Unlock()
}

func (p *Proxy) wsReader() {
	for {
		select {
		default:
			_, msg, err := p.ws.ReadMessage()
			if err != nil {
				p.Shutdown()
				return
			}

			_, err = p.ics.Write(msg)
			if err != nil {
				p.Shutdown()
				return
			}
		case <-p.wsReaderChannel:
			return
		}
	}
}

func (p *Proxy) icsReader() {
	for {
		select {
		default:
			msg, err := p.ics.ReadUntil("\n")
			if err != nil {
				p.Shutdown()
				return
			}

			p.Lock()
			err = p.ws.WriteMessage(websocket.TextMessage, msg)
			p.Unlock()
			if err != nil {
				p.Shutdown()
				return
			}
		case <-p.icsReaderChannel:
			return
		}
	}
}

func (p *Proxy) keepAlive() {
	var lastResponse int64
	atomic.StoreInt64(&lastResponse, time.Now().UnixNano())
	p.ws.SetPongHandler(func(msg string) error {
		atomic.StoreInt64(&lastResponse, time.Now().UnixNano())
		return nil
	})

	for {
		select {
		default:
			p.Lock()
			err := p.ws.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			p.Unlock()
			if err != nil {
				p.Shutdown()
				return
			}
			time.Sleep(keepAlive / 2)
			if atomic.LoadInt64(&lastResponse) < time.Now().Add(-timeout).UnixNano() {
				p.Shutdown()
				return
			}
		case <-p.keepAliveChannel:
			return
		}
	}
}
