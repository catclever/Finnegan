package publisher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"Finnegan/modules/atproto/config"
)

type Session struct {
	DID             string `json:"did"`
	Handle          string `json:"handle"`
	Email           string `json:"email"`
	AccessJwt       string `json:"accessJwt"`
	RefreshJwt      string `json:"refreshJwt"`
	Active          bool   `json:"active"`
}

type AuthManager struct {
	Config  *config.Config
	Session *Session
	Client  *http.Client
}

func NewAuthManager(cfg *config.Config) *AuthManager {
	return &AuthManager{
		Config: cfg,
		Client: &http.Client{},
	}
}

func (a *AuthManager) CreateSession() error {
	url := fmt.Sprintf("%s/xrpc/com.atproto.server.createSession", a.Config.GetPDSUrl())

	// Identifier can be either handle or DID.
	identifier := a.Config.Handle
	if identifier == "" {
		identifier = a.Config.DID
	}

	payload := map[string]string{
		"identifier": identifier,
		"password":   a.Config.Password,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create session: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return err
	}

	session.Active = true
	a.Session = &session
	return nil
}

func (a *AuthManager) RefreshSession() error {
	if a.Session == nil || a.Session.RefreshJwt == "" {
		return fmt.Errorf("no active session to refresh")
	}

	url := fmt.Sprintf("%s/xrpc/com.atproto.server.refreshSession", a.Config.GetPDSUrl())

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+a.Session.RefreshJwt)

	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to refresh session: status %d", resp.StatusCode)
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return err
	}

	session.Active = true
	a.Session = &session
	return nil
}

func (a *AuthManager) DoAuthenticatedRequest(req *http.Request) (*http.Response, error) {
	if a.Session == nil || !a.Session.Active {
		if err := a.CreateSession(); err != nil {
			return nil, err
		}
	}

	// Clone the request for a potential retry
	reqClone := req.Clone(req.Context())

	// Try request with current AccessJwt
	req.Header.Set("Authorization", "Bearer "+a.Session.AccessJwt)
	resp, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}

	// If token expired, refresh and retry
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		if err := a.RefreshSession(); err != nil {
			// If refresh fails, try re-logging in
			if err = a.CreateSession(); err != nil {
				return nil, fmt.Errorf("failed to authenticate: %w", err)
			}
		}

		// Retry with new token
		reqClone.Header.Set("Authorization", "Bearer "+a.Session.AccessJwt)
		return a.Client.Do(reqClone)
	}

	return resp, nil
}
