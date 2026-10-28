package publisher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"Finnegan/modules/atproto/config"
)

type SessionManager struct {
	Config     *config.Config
	accessJwt  string
	refreshJwt string
	did        string
	mu         sync.Mutex
}

func NewSessionManager(cfg *config.Config) *SessionManager {
	if cfg.PDSUrl == "" {
		cfg.PDSUrl = "https://bsky.social"
	}
	return &SessionManager{
		Config: cfg,
	}
}

type SessionResponse struct {
	AccessJwt  string `json:"accessJwt"`
	RefreshJwt string `json:"refreshJwt"`
	Did        string `json:"did"`
}

func (sm *SessionManager) CreateSession() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	identifier := sm.Config.Handle
	if identifier == "" {
		identifier = sm.Config.DID
	}
	password := sm.Config.AppPassword
	if password == "" {
		password = sm.Config.Password
	}

	payload := map[string]string{
		"identifier": identifier,
		"password":   password,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := sm.Config.PDSUrl + "/xrpc/com.atproto.server.createSession"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("createSession failed: %d %s", resp.StatusCode, string(body))
	}

	var sess SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
		return err
	}

	sm.accessJwt = sess.AccessJwt
	sm.refreshJwt = sess.RefreshJwt
	sm.did = sess.Did
	return nil
}

func (sm *SessionManager) RefreshSession() error {
	sm.mu.Lock()
	refreshJwt := sm.refreshJwt
	sm.mu.Unlock()

	if refreshJwt == "" {
		return sm.CreateSession()
	}

	url := sm.Config.PDSUrl + "/xrpc/com.atproto.server.refreshSession"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+refreshJwt)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If refresh fails, try a full create
		return sm.CreateSession()
	}

	var sess SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
		return err
	}

	sm.mu.Lock()
	sm.accessJwt = sess.AccessJwt
	sm.refreshJwt = sess.RefreshJwt
	sm.did = sess.Did
	sm.mu.Unlock()
	return nil
}

// DoRequest makes an authenticated request, auto-refreshing token on 401.
func (sm *SessionManager) DoRequest(req *http.Request) (*http.Response, error) {
	sm.mu.Lock()
	accessJwt := sm.accessJwt
	sm.mu.Unlock()

	if accessJwt == "" {
		if err := sm.CreateSession(); err != nil {
			return nil, err
		}
		sm.mu.Lock()
		accessJwt = sm.accessJwt
		sm.mu.Unlock()
	}

	req.Header.Set("Authorization", "Bearer "+accessJwt)

	// Since we might need to retry, we must be able to read req.Body again.
	// But let's assume req.Body is small and can be cached if needed, or we just fail and let caller retry.
	// We'll read the request body into a buffer if it exists so we can recreate it.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := sm.RefreshSession(); err != nil {
			return nil, err
		}

		sm.mu.Lock()
		accessJwt = sm.accessJwt
		sm.mu.Unlock()

		req.Header.Set("Authorization", "Bearer "+accessJwt)
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		resp, err = http.DefaultClient.Do(req)
	}

	return resp, err
}

func (sm *SessionManager) GetDID() string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.did
}
