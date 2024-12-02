package gqltest

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
)

type HeaderFunc func(h *http.Header)

func WsConnect(url string, hfs ...HeaderFunc) *websocket.Conn {
	return WsConnectWithSubprocotol(url, "", hfs...)
}

func WsConnectWithSubprocotol(url, subprocotol string, hfs ...HeaderFunc) *websocket.Conn {
	h := make(http.Header)
	if subprocotol != "" {
		h.Add("Sec-WebSocket-Protocol", subprocotol)
	}
	for _, hf := range hfs {
		hf(&h)
	}
	c, resp, err := websocket.DefaultDialer.Dial(strings.ReplaceAll(url, "http://", "ws://"), h)
	if err != nil {
		panic(err)
	}
	_ = resp.Body.Close()

	return c
}

type OperationMessage struct {
	Payload json.RawMessage `json:"payload,omitempty"`
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
}

const (
	ConnectionInitMsg      = "connection_init"      // Client -> Server
	ConnectionTerminateMsg = "connection_terminate" // Client -> Server
	StartMsg               = "start"                // Client -> Server
	StopMsg                = "stop"                 // Client -> Server
	ConnectionAckMsg       = "connection_ack"       // Server -> Client
	ConnectionErrorMsg     = "connection_error"     // Server -> Client
	DataMsg                = "data"                 // Server -> Client
	ErrorMsg               = "error"                // Server -> Client
	CompleteMsg            = "complete"             // Server -> Client
	ConnectionKeepAliveMsg = "ka"                   // Server -> Client
)

// copied out from websocket_graphql_transport_ws.go to keep these private

const (
	GraphqltransportwsSubprotocol = "graphql-transport-ws"

	GraphqltransportwsConnectionInitMsg = "connection_init"
	GraphqltransportwsConnectionAckMsg  = "connection_ack"
	GraphqltransportwsSubscribeMsg      = "subscribe"
	GraphqltransportwsNextMsg           = "next"
	GraphqltransportwsCompleteMsg       = "complete"
	GraphqltransportwsPingMsg           = "ping"
	GraphqltransportwsPongMsg           = "pong"
)

// ReadOp reads an operation message from the websocket connection
func ReadOp(conn *websocket.Conn) OperationMessage {
	var msg OperationMessage
	if err := conn.ReadJSON(&msg); err != nil {
		panic(err)
	}
	return msg
}
