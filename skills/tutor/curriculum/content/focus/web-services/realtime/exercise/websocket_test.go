package realtime

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The example from RFC 6455 §1.3. If your implementation reproduces it, every
// browser on earth will accept your handshake.
func TestAcceptKeyMatchesTheRFCExample(t *testing.T) {
	const (
		key  = "dGhlIHNhbXBsZSBub25jZQ=="
		want = "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	)
	if got := AcceptKey(key); got != want {
		t.Errorf("AcceptKey(%q) = %q, want %q", key, got, want)
	}
}

func upgradeRequest() *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/ws", nil)
	r.Header.Set("Connection", "keep-alive, Upgrade")
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Sec-WebSocket-Version", "13")
	r.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	return r
}

func TestCheckUpgrade(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*http.Request)
		wantErr error
	}{
		{"a valid handshake", func(*http.Request) {}, nil},
		{
			name:   "Connection is a comma-separated list, matched case-insensitively",
			mutate: func(r *http.Request) { r.Header.Set("Connection", "Upgrade") },
		},
		{
			name:   "Upgrade is case-insensitive",
			mutate: func(r *http.Request) { r.Header.Set("Upgrade", "WebSocket") },
		},
		{
			name:    "POST is not an upgrade",
			mutate:  func(r *http.Request) { r.Method = http.MethodPost },
			wantErr: ErrBadMethod,
		},
		{
			name:    "no Connection: Upgrade",
			mutate:  func(r *http.Request) { r.Header.Set("Connection", "keep-alive") },
			wantErr: ErrNotUpgrade,
		},
		{
			name:    "some other protocol",
			mutate:  func(r *http.Request) { r.Header.Set("Upgrade", "h2c") },
			wantErr: ErrNotUpgrade,
		},
		{
			name:    "a version nobody speaks any more",
			mutate:  func(r *http.Request) { r.Header.Set("Sec-WebSocket-Version", "8") },
			wantErr: ErrBadVersion,
		},
		{
			name:    "the key is not base64",
			mutate:  func(r *http.Request) { r.Header.Set("Sec-WebSocket-Key", "not base64!") },
			wantErr: ErrBadKey,
		},
		{
			name:    "the key is base64 of the wrong length",
			mutate:  func(r *http.Request) { r.Header.Set("Sec-WebSocket-Key", "c2hvcnQ=") },
			wantErr: ErrBadKey,
		},
		{
			name:    "no key at all",
			mutate:  func(r *http.Request) { r.Header.Del("Sec-WebSocket-Key") },
			wantErr: ErrBadKey,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := upgradeRequest()
			tc.mutate(r)
			err := CheckUpgrade(r)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("CheckUpgrade() = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("CheckUpgrade() = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestHandshakeSwitchesProtocols(t *testing.T) {
	u := Upgrader{Origins: []string{"https://app.example.com"}}
	r := upgradeRequest()
	r.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	if err := u.Handshake(rec, r); err != nil {
		t.Fatalf("Handshake() = %v, want nil", err)
	}
	if rec.Code != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", rec.Code)
	}
	for header, want := range map[string]string{
		"Upgrade":              "websocket",
		"Connection":           "Upgrade",
		"Sec-WebSocket-Accept": "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=",
	} {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

// A browser sends the user's cookies with a websocket handshake and CORS does
// not apply, so the Origin check is the whole of the defence against
// cross-site websocket hijacking.
func TestHandshakeChecksOrigin(t *testing.T) {
	tests := []struct {
		name       string
		origins    []string
		origin     string
		wantStatus int
		wantErr    error
	}{
		{"an allowed origin", []string{"https://app.example.com"}, "https://app.example.com", http.StatusSwitchingProtocols, nil},
		{"a hostile origin", []string{"https://app.example.com"}, "https://evil.example", http.StatusForbidden, ErrBadOrigin},
		{"a suffix that is not a match", []string{"https://app.example.com"}, "https://app.example.com.evil.test", http.StatusForbidden, ErrBadOrigin},
		{"deny by default", nil, "https://app.example.com", http.StatusForbidden, ErrBadOrigin},
		{"no Origin at all is not a browser", []string{"https://app.example.com"}, "", http.StatusSwitchingProtocols, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := upgradeRequest()
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()

			err := Upgrader{Origins: tc.origins}.Handshake(rec, r)
			if tc.wantErr == nil && err != nil {
				t.Fatalf("Handshake() = %v, want nil", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("Handshake() = %v, want %v", err, tc.wantErr)
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

func TestHandshakeReportsWhatTheClientGotWrong(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*http.Request)
		wantStatus int
		wantHeader [2]string
	}{
		{
			name:       "POST",
			mutate:     func(r *http.Request) { r.Method = http.MethodPost },
			wantStatus: http.StatusMethodNotAllowed,
			wantHeader: [2]string{"Allow", "GET"},
		},
		{
			name:       "an old protocol version",
			mutate:     func(r *http.Request) { r.Header.Set("Sec-WebSocket-Version", "8") },
			wantStatus: http.StatusUpgradeRequired,
			wantHeader: [2]string{"Sec-WebSocket-Version", "13"},
		},
		{
			name:       "an ordinary request",
			mutate:     func(r *http.Request) { r.Header.Del("Upgrade") },
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := upgradeRequest()
			tc.mutate(r)
			rec := httptest.NewRecorder()

			u := Upgrader{Origins: []string{"https://app.example.com"}}
			if err := u.Handshake(rec, r); err == nil {
				t.Fatal("Handshake() = nil, want an error")
			}
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantHeader[0] != "" {
				if got := rec.Header().Get(tc.wantHeader[0]); got != tc.wantHeader[1] {
					t.Errorf("%s = %q, want %q", tc.wantHeader[0], got, tc.wantHeader[1])
				}
			}
			if got := rec.Header().Get("Sec-WebSocket-Accept"); got != "" {
				t.Errorf("Sec-WebSocket-Accept = %q on a rejected handshake, want it unset", got)
			}
		})
	}
}
