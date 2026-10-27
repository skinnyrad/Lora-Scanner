package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/skinnyrad/Lora-Scanner/internal/frames"
	"github.com/skinnyrad/Lora-Scanner/internal/gateway"
	"github.com/skinnyrad/Lora-Scanner/internal/hub"
	"github.com/skinnyrad/Lora-Scanner/internal/serial"
)

func newServer(t *testing.T) (http.Handler, *frames.Store) {
	h := hub.New()
	st := frames.NewStore(h, 0)
	cfg := gateway.NewConfig(filepath.Join(t.TempDir(), "c.json"))
	return (&Server{
		Serial: serial.NewManager(h, st), Hub: h, Frames: st, Config: cfg,
		Scraper: gateway.NewScraper(cfg, st, "u", "p"),
		UI: fstest.MapFS{
			"index.html":    {Data: []byte("<html>spa</html>")},
			"assets/app.js": {Data: []byte("console.log(1)")},
		},
	}).Router(), st
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestRouting(t *testing.T) {
	h, _ := newServer(t)
	for _, p := range []string{"/", "/survey", "/some/deep/link"} {
		if w := do(h, "GET", p, ""); w.Code != 200 || !strings.Contains(w.Body.String(), "spa") {
			t.Errorf("%s: %d %s", p, w.Code, w.Body)
		}
	}
	if w := do(h, "GET", "/assets/app.js", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "console") {
		t.Errorf("asset: %d %s", w.Code, w.Body)
	}
	if w := do(h, "GET", "/api/nope", ""); w.Code != 404 || !strings.Contains(w.Header().Get("Content-Type"), "json") {
		t.Errorf("api 404: %d %s", w.Code, w.Body)
	}
}

func TestAPIBasics(t *testing.T) {
	h, _ := newServer(t)
	if w := do(h, "GET", "/api/transceivers", ""); w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Errorf("list: %d %s", w.Code, w.Body)
	}
	if w := do(h, "POST", "/api/transceivers/1/transmit", `{"text":"x"}`); w.Code != 404 {
		t.Errorf("transmit unknown: %d", w.Code)
	}
	if w := do(h, "DELETE", "/api/transceivers/1", ""); w.Code != 404 {
		t.Errorf("remove unknown: %d", w.Code)
	}
	if w := do(h, "POST", "/api/transceivers", `{"port":""}`); w.Code != 500 && w.Code != 400 {
		t.Errorf("empty port: %d", w.Code)
	}
	if w := do(h, "GET", "/api/serial", ""); w.Code != 404 {
		t.Errorf("old serial routes should be gone: %d", w.Code)
	}
	if w := do(h, "POST", "/api/gateways", `{"ip":"nope"}`); w.Code != 400 {
		t.Errorf("bad ip: %d", w.Code)
	}
	if w := do(h, "POST", "/api/gateways", `{"ip":"10.0.0.1"}`); w.Code != 201 {
		t.Errorf("add: %d %s", w.Code, w.Body)
	}
	if w := do(h, "POST", "/api/gateways", `{"ip":"10.0.0.1"}`); w.Code != 409 {
		t.Errorf("duplicate: %d", w.Code)
	}
	if w := do(h, "GET", "/api/gateways", ""); strings.TrimSpace(w.Body.String()) != `["10.0.0.1"]` {
		t.Errorf("list: %s", w.Body)
	}
	if w := do(h, "GET", "/api/gateways/poll", ""); strings.TrimSpace(w.Body.String()) != `{"seconds":30}` {
		t.Errorf("poll get: %s", w.Body)
	}
	if w := do(h, "PUT", "/api/gateways/poll", `{"seconds":2}`); w.Code != 400 {
		t.Errorf("poll too small: %d", w.Code)
	}
	if w := do(h, "PUT", "/api/gateways/poll", `{"seconds":120}`); w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"seconds":120}` {
		t.Errorf("poll put: %d %s", w.Code, w.Body)
	}
	if w := do(h, "GET", "/api/gateways/status", ""); !strings.Contains(w.Body.String(), `"10.0.0.1":{"state":"unknown","lastSuccess":null}`) {
		t.Errorf("status before first poll: %s", w.Body)
	}
	if w := do(h, "DELETE", "/api/gateways/10.0.0.9", ""); w.Code != 404 {
		t.Errorf("remove unknown: %d", w.Code)
	}
	if w := do(h, "DELETE", "/api/gateways/10.0.0.1", ""); w.Code != 204 {
		t.Errorf("remove: %d", w.Code)
	}
	if w := do(h, "GET", "/api/survey", ""); w.Code != 404 {
		t.Errorf("survey route should be gone: %d", w.Code)
	}
	if w := do(h, "POST", "/api/transceivers/1/beacon", `{"text":"x","intervalMs":10}`); w.Code != 400 {
		t.Errorf("beacon interval: %d", w.Code)
	}
}

func TestFramesAndDevices(t *testing.T) {
	h, st := newServer(t)
	rssi := -50
	st.Add(frames.Frame{Kind: frames.KindRaw, Info: "noise"})
	st.Add(frames.Frame{Kind: frames.KindLoRa, Identifier: "dev1", Freq: 915, RSSI: &rssi, Info: "hi"})

	w := do(h, "GET", "/api/frames", "")
	var fs []frames.Frame
	if err := json.Unmarshal(w.Body.Bytes(), &fs); err != nil || len(fs) != 2 || fs[1].Identifier != "dev1" {
		t.Fatalf("frames: %d %s", w.Code, w.Body)
	}
	w = do(h, "GET", "/api/frames?after=1", "")
	if err := json.Unmarshal(w.Body.Bytes(), &fs); err != nil || len(fs) != 1 || fs[0].ID != 2 {
		t.Fatalf("after: %s", w.Body)
	}
	if w := do(h, "GET", "/api/frames?after=x", ""); w.Code != 400 {
		t.Errorf("bad after: %d", w.Code)
	}
	w = do(h, "GET", "/api/devices", "")
	var ds []frames.Device
	if err := json.Unmarshal(w.Body.Bytes(), &ds); err != nil || len(ds) != 1 || ds[0].Frames != 1 || ds[0].Identifier != "dev1" {
		t.Fatalf("devices: %s", w.Body)
	}
	if w := do(h, "GET", "/api/frames.csv", ""); w.Code != 200 || !strings.HasPrefix(w.Body.String(), "No,Time,Kind,Source,Identifier,Frequency,RSSI,Info") {
		t.Errorf("csv: %s", w.Body)
	}
	if w := do(h, "DELETE", "/api/frames", ""); w.Code != 204 {
		t.Errorf("clear: %d", w.Code)
	}
	if w := do(h, "GET", "/api/devices", ""); strings.TrimSpace(w.Body.String()) != "[]" {
		t.Errorf("devices after clear: %s", w.Body)
	}
}
