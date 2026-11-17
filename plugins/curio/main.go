package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed ui.js
var uiBundle []byte

type pluginManifest struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	ThemeColor  string         `json:"themeColor"`
	CardTypes   []cardTypeSpec `json:"cardTypes"`
	Commands    []commandDef   `json:"commands"`
}

type cardTypeSpec struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Prefix      string `json:"prefix"`
	SubmitLabel string `json:"submitLabel"`
	BadgeColor  string `json:"badgeColor"`
	Selectable  bool   `json:"selectable"`
	UIEntry     string `json:"uiEntry"`
}

type paramSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type commandDef struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Trigger     string      `json:"trigger"`
	Description string      `json:"description"`
	Namespace   string      `json:"namespace"`
	IsPrimary   bool        `json:"isPrimary"`
	Params      []paramSpec `json:"params,omitempty"`
}

type pluginRequest struct {
	Method string                 `json:"method"`
	Type   string                 `json:"type"`
	ID     string                 `json:"id"`
	Params map[string]interface{} `json:"params"`
	LLMs   map[string]llmConfig   `json:"llms"`
	Config map[string]interface{} `json:"config"`
}

func buildManifest() pluginManifest {
	return pluginManifest{
		Name:        "curio",
		Version:     "1.0.0",
		Description: "商品网页快照与恋物藏品 (Curio 外部插件驱动)",
		ThemeColor:  "#10b981",
		CardTypes: []cardTypeSpec{
			{
				Type:        "curio",
				Name:        "奇物标本",
				Description: "商品网页快照与恋物藏品 (Curio 外部插件驱动)",
				Icon:        "🏺",
				Prefix:      ":curio",
				SubmitLabel: "标本入库",
				BadgeColor:  "#10b981",
				Selectable:  true,
				UIEntry:     "/api/plugins/curio/ui.js",
			},
		},
		Commands: []commandDef{
			{
				ID:          "curio:ingest",
				Name:        "奇物截图入库",
				Trigger:     "curio",
				Description: "从本地截图路径提取奇物标本卡片",
				Namespace:   "curio",
				IsPrimary:   true,
				Params: []paramSpec{
					{
						Name:        "screenshot",
						Type:        "text",
						Description: "截图文件路径",
					},
				},
			},
		},
	}
}

func handleCaptureCard(ctx context.Context, req pluginRequest) (*captureResult, error) {
	var capReq captureRequest
	if req.Params != nil {
		b, err := json.Marshal(req.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		if err := json.Unmarshal(b, &capReq); err != nil {
			return nil, fmt.Errorf("failed to decode capture request: %w", err)
		}
	}

	cleanText := capReq.CleanText
	if cleanText == "" && strings.TrimSpace(capReq.Text) != "" && len(capReq.Tags) == 0 {
		cleanText = strings.TrimSpace(capReq.Text)
	}

	var decodedImgs []decodedImage
	for _, img := range capReq.Images {
		if strings.TrimSpace(img.Data) == "" {
			continue
		}
		d, err := decodeCaptureImage(img)
		if err != nil {
			return nil, err
		}
		decodedImgs = append(decodedImgs, d)
	}

	if len(decodedImgs) > 0 {
		homeDir := resolveHomeDir(capReq.HomeDir)
		return ingestCurioImages(ctx, homeDir, req.LLMs, decodedImgs, cleanText, capReq.Tags)
	}

	if strings.TrimSpace(capReq.Text) == "" && strings.TrimSpace(cleanText) == "" {
		return nil, errors.New("empty text and images")
	}

	return captureTextOnly(cleanText, capReq.Tags)
}

func handleExecuteCommand(ctx context.Context, req pluginRequest) (map[string]interface{}, error) {
	if req.ID != "curio:ingest" {
		return nil, fmt.Errorf("unknown command: %s", req.ID)
	}

	var screenshotPath, note, explicitHome string
	var tags []string

	if req.Params != nil {
		if s, ok := req.Params["screenshot"].(string); ok {
			screenshotPath = strings.TrimSpace(s)
		}
		if screenshotPath == "" {
			if s, ok := req.Params["text"].(string); ok {
				screenshotPath = strings.TrimSpace(s)
			} else if s, ok := req.Params["content"].(string); ok {
				screenshotPath = strings.TrimSpace(s)
			}
		}
		if n, ok := req.Params["note"].(string); ok {
			note = strings.TrimSpace(n)
		}
		if h, ok := req.Params["homeDir"].(string); ok {
			explicitHome = strings.TrimSpace(h)
		}
		if rawTags, ok := req.Params["tags"].([]interface{}); ok {
			for _, rt := range rawTags {
				if ts, ok := rt.(string); ok && strings.TrimSpace(ts) != "" {
					tags = append(tags, strings.TrimSpace(ts))
				}
			}
		}
	}

	if screenshotPath == "" {
		return nil, errors.New("screenshot path is required for curio:ingest")
	}

	imgBytes, err := os.ReadFile(screenshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read screenshot %s: %w", screenshotPath, err)
	}

	ext := strings.ToLower(filepath.Ext(screenshotPath))
	if ext == "" {
		ext = ".png"
	}
	if ext == ".jpeg" {
		ext = ".jpg"
	}

	homeDir := resolveHomeDir(explicitHome)
	if homeDir == "" {
		return nil, errors.New("homeDir or FINNEGAN_HOME is required for curio:ingest")
	}

	img := decodedImage{
		Data:     imgBytes,
		Ext:      ext,
		MimeType: extToMimeType(ext),
	}

	capRes, err := ingestCurioImages(ctx, homeDir, req.LLMs, []decodedImage{img}, note, tags)
	if err != nil {
		return nil, err
	}

	cardPath, err := writeCurioMarkdownFile(homeDir, capRes)
	if err != nil {
		return nil, err
	}
	capRes.FilePath = cardPath

	return map[string]interface{}{
		"status":   "ok",
		"docId":    capRes.DocID,
		"filePath": cardPath,
		"title":    capRes.Title,
		"images":   capRes.Images,
		"meta":     capRes.Meta,
	}, nil
}

func runCLI(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) > 1 {
		switch args[1] {
		case "--finnegan-manifest":
			enc := json.NewEncoder(stdout)
			enc.SetEscapeHTML(false)
			if err := enc.Encode(buildManifest()); err != nil {
				fmt.Fprintf(stderr, "failed to encode manifest: %v\n", err)
				return 1
			}
			return 0
		case "--finnegan-ui":
			if _, err := stdout.Write(uiBundle); err != nil {
				fmt.Fprintf(stderr, "failed to write ui bundle: %v\n", err)
				return 1
			}
			return 0
		}
	}

	var req pluginRequest
	if err := json.NewDecoder(stdin).Decode(&req); err != nil {
		fmt.Fprintf(stderr, "invalid request json: %v\n", err)
		return 1
	}

	method := req.Method
	if method == "" {
		method = req.Type
	}

	ctx := context.Background()
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)

	switch method {
	case "capture_card":
		res, err := handleCaptureCard(ctx, req)
		if err != nil {
			_ = enc.Encode(map[string]interface{}{"error": err.Error()})
			return 0
		}
		_ = enc.Encode(map[string]interface{}{"result": res})
		return 0

	case "execute_command":
		res, err := handleExecuteCommand(ctx, req)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			_ = enc.Encode(map[string]interface{}{"error": err.Error()})
			return 1
		}
		_ = enc.Encode(map[string]interface{}{"result": res})
		return 0

	case "execute_action":
		errMsg := fmt.Sprintf("unknown action: %s", req.ID)
		fmt.Fprintf(stderr, "%s\n", errMsg)
		_ = enc.Encode(map[string]interface{}{"error": errMsg})
		return 1

	default:
		errMsg := fmt.Sprintf("unsupported method: %s", method)
		fmt.Fprintf(stderr, "%s\n", errMsg)
		_ = enc.Encode(map[string]interface{}{"error": errMsg})
		return 1
	}
}

func main() {
	os.Exit(runCLI(os.Args, os.Stdin, os.Stdout, os.Stderr))
}
