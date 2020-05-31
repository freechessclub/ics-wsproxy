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
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/websocket"
)

const (
	// default ICS server address to connect to
	serverAddr = "freechess.org:5000"
)

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "unable to upgrade to websockets", http.StatusBadRequest)
		return
	}
	ws.SetReadLimit(2048)

	_, err = NewProxy(serverAddr, ws)
	if err != nil {
		http.Error(w, "unable to create new proxy", http.StatusInternalServerError)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Println("Using default port 8080")
	}

	http.HandleFunc("/ws", handleWebsocket)
	loggingRouter := handlers.LoggingHandler(os.Stdout, http.DefaultServeMux)
	log.Println(http.ListenAndServe(":"+port, loggingRouter))
}
