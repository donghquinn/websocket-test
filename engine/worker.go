package engine

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// envelope is the wire format used when Echo is set: the client stamps a
// send timestamp and expects the server to echo the message back verbatim,
// so the read loop can recover it and compute round-trip latency.
type envelope struct {
	ID   uint64 `json:"id"`
	TS   int64  `json:"ts"`
	Data string `json:"data"`
}

const payloadCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomPayload(size int, rng *rand.Rand) []byte {
	b := make([]byte, size)
	for i := range b {
		b[i] = payloadCharset[rng.Intn(len(payloadCharset))]
	}
	return b
}

// RunWorker drives a single simulated client connection until ctx is done or
// the connection errors out. It blocks until the connection closes.
func RunWorker(ctx context.Context, cfg *Config, st *Stats, headers http.Header) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	basePayload := []byte(cfg.Payload)
	if len(basePayload) == 0 {
		basePayload = randomPayload(cfg.PayloadSize, rng)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: cfg.ConnectTimeout,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: cfg.Insecure},
	}
	if cfg.Subprotocol != "" {
		dialer.Subprotocols = []string{cfg.Subprotocol}
	}

	st.ConnectAttempt()
	conn, _, err := dialer.DialContext(ctx, cfg.URL, headers)
	if err != nil {
		st.ConnectFail()
		return
	}
	st.ConnectSuccess()
	st.ConnActive(1)
	defer func() {
		conn.Close()
		st.ConnActive(-1)
		st.Disconnect()
	}()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			st.MsgRecv(len(msg))
			if cfg.Echo {
				var env envelope
				if json.Unmarshal(msg, &env) == nil && env.TS != 0 {
					st.Latency(time.Since(time.Unix(0, env.TS)))
				}
			}
		}
	}()

	var tickerC <-chan time.Time
	if cfg.Rate > 0 {
		ticker := time.NewTicker(time.Duration(float64(time.Second) / cfg.Rate))
		defer ticker.Stop()
		tickerC = ticker.C
	}

	opcode := websocket.TextMessage
	if cfg.Binary && !cfg.Echo {
		opcode = websocket.BinaryMessage
	}

	var msgID uint64
	for {
		select {
		case <-ctx.Done():
			deadline := time.Now().Add(time.Second)
			conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), deadline)
			return
		case <-readDone:
			st.Error()
			return
		case <-tickerC:
			msgID++
			data := basePayload
			if cfg.Echo {
				env := envelope{ID: msgID, TS: time.Now().UnixNano(), Data: string(basePayload)}
				encoded, err := json.Marshal(env)
				if err != nil {
					st.Error()
					continue
				}
				data = encoded
			}
			conn.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
			if err := conn.WriteMessage(opcode, data); err != nil {
				st.Error()
				return
			}
			st.MsgSent(len(data))
		}
	}
}
