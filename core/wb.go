package core

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	WbBase = "https://stream.wb.ru"
	WbUA   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

type TurnCred struct{ URL, User, Pass string }

type CredsCache struct {
	sync.Mutex
	creds []TurnCred
	at    time.Time
}

func (c *CredsCache) Get(force bool) ([]TurnCred, error) {
	c.Lock()
	defer c.Unlock()
	if !force && len(c.creds) > 0 && time.Since(c.at) < CredsTTL {
		return c.creds, nil
	}
	cr, err := FetchCreds()
	if err != nil {
		return nil, err
	}
	c.creds, c.at = cr, time.Now()
	return cr, nil
}

func wbReq(cl *http.Client, method, ep string, body []byte, tok string) ([]byte, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	rq, _ := http.NewRequest(method, WbBase+ep, rd)
	rq.Header.Set("User-Agent", WbUA)
	rq.Header.Set("Accept", "application/json")
	rq.Header.Set("Accept-Language", "en-US,en;q=0.9")
	rq.Header.Set("Origin", WbBase)
	rq.Header.Set("Referer", WbBase+"/")
	if body != nil {
		rq.Header.Set("Content-Type", "application/json")
	}
	if tok != "" {
		rq.Header.Set("Authorization", "Bearer "+tok)
	}
	rs, err := cl.Do(rq)
	if err != nil {
		return nil, err
	}
	defer rs.Body.Close()
	var r io.Reader = rs.Body
	if rs.Header.Get("Content-Encoding") == "gzip" {
		if g, e := gzip.NewReader(rs.Body); e == nil {
			defer g.Close()
			r = g
		}
	}
	b, _ := io.ReadAll(r)
	if rs.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", rs.StatusCode, string(b))
	}
	return b, nil
}

func FetchCreds() ([]TurnCred, error) {
	log := getLog()
	cl := &http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{}},
	}
	nm := fmt.Sprintf("lh_%d", time.Now().UnixMilli()%100000)

	log.Info("Connecting to WB Stream...")

	// 1. guest register
	rr, err := wbReq(cl, "POST", "/auth/api/v1/auth/user/guest-register",
		[]byte(`{"displayName":"`+nm+`"}`), "")
	if err != nil {
		return nil, fmt.Errorf("guest register: %w", err)
	}
	var reg struct {
		AccessToken string `json:"accessToken"`
	}
	json.Unmarshal(rr, &reg)
	if reg.AccessToken == "" {
		return nil, fmt.Errorf("no access token")
	}
	log.Info("Guest registered")

	// 2. create room
	rr, err = wbReq(cl, "POST", "/api-room/api/v2/room",
		[]byte(`{"roomType":"ROOM_TYPE_ALL_ON_SCREEN","roomPrivacy":"ROOM_PRIVACY_FREE"}`),
		reg.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}
	var room struct {
		RoomID string `json:"roomId"`
	}
	json.Unmarshal(rr, &room)
	if room.RoomID == "" {
		return nil, fmt.Errorf("no room ID")
	}
	log.Info(fmt.Sprintf("Room created: %s", room.RoomID[:min(8, len(room.RoomID))]))

	// 3. join
	wbReq(cl, "POST", fmt.Sprintf("/api-room/api/v1/room/%s/join", room.RoomID),
		[]byte("{}"), reg.AccessToken)

	// 4. token
	rr, err = wbReq(cl, "GET", fmt.Sprintf(
		"/api-room-manager/api/v1/room/%s/token?deviceType=PARTICIPANT_DEVICE_TYPE_WEB_DESKTOP&displayName=%s",
		room.RoomID, url.QueryEscape(nm)), nil, reg.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}
	var tok struct {
		RoomToken string `json:"roomToken"`
	}
	json.Unmarshal(rr, &tok)
	if tok.RoomToken == "" {
		return nil, fmt.Errorf("no room token")
	}

	// 5. LiveKit ICE
	log.Info("Negotiating ICE (LiveKit)...")
	creds, err := lkICE(tok.RoomToken)
	if err != nil {
		return nil, fmt.Errorf("livekit ICE: %w", err)
	}
	for _, c := range creds {
		log.Info(fmt.Sprintf("  → %s", c.URL))
	}
	return creds, nil
}

func lkICE(token string) ([]TurnCred, error) {
	u := "wss://wbstream01-el.wb.ru:7880/rtc?access_token=" + url.QueryEscape(token) +
		"&auto_subscribe=1&sdk=js&version=2.15.3&protocol=16&adaptive_stream=1"
	conn, _, err := (&websocket.Dialer{
		TLSClientConfig:  &tls.Config{},
		HandshakeTimeout: 10 * time.Second,
	}).Dial(u, http.Header{
		"User-Agent": {WbUA},
		"Origin":     {WbBase},
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	for i := 0; i < 15; i++ {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if c := PbICE(msg); len(c) > 0 {
			return Dedup(c), nil
		}
	}
	return nil, fmt.Errorf("TURN not found in LiveKit response")
}

func Dedup(cc []TurnCred) (r []TurnCred) {
	seen := map[string]bool{}
	for _, c := range cc {
		k := c.URL + "|" + c.User
		if !seen[k] {
			seen[k] = true
			r = append(r, c)
		}
	}
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
