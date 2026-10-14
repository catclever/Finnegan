package main

import (
	"embed"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"Finnegan/core/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func getContentType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".avif":
		return "image/avif"
	case ".bmp":
		return "image/bmp"
	case ".ico":
		return "image/x-icon"
	default:
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
		f, err := os.Open(filePath)
		if err == nil {
			defer f.Close()
			buf := make([]byte, 512)
			n, _ := f.Read(buf)
			if n > 0 {
				return http.DetectContentType(buf[:n])
			}
		}
		return "application/octet-stream"
	}
}

func createAssetHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/assets/") {
			http.NotFound(w, r)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		home := app.Home
		if home == "" {
			_, h, err := config.LoadConfig()
			if err == nil {
				home = h
				app.Home = h
			}
		}
		if home == "" {
			http.Error(w, "Home directory not set", http.StatusInternalServerError)
			return
		}

		assetsDir := filepath.Join(home, "assets")
		relPath := strings.TrimPrefix(r.URL.Path, "/api/assets/")
		relPath = filepath.Clean(relPath)
		if relPath == "." || relPath == "" || strings.HasPrefix(relPath, "..") || strings.HasPrefix(relPath, "/") || strings.HasPrefix(relPath, "\\") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		fullPath := filepath.Join(assetsDir, relPath)
		cleanAssetsDir, err := filepath.Abs(assetsDir)
		if err != nil {
			cleanAssetsDir = filepath.Clean(assetsDir)
		}
		cleanFullPath, err := filepath.Abs(fullPath)
		if err != nil {
			cleanFullPath = filepath.Clean(fullPath)
		}

		if !strings.HasPrefix(cleanFullPath, cleanAssetsDir+string(filepath.Separator)) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		fi, err := os.Stat(cleanFullPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if fi.IsDir() {
			http.NotFound(w, r)
			return
		}

		realPath, err := filepath.EvalSymlinks(cleanFullPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		realAssetsDir, err := filepath.EvalSymlinks(cleanAssetsDir)
		if err != nil {
			realAssetsDir = cleanAssetsDir
		}
		if !strings.HasPrefix(realPath, realAssetsDir+string(filepath.Separator)) && realPath != realAssetsDir {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		f, err := os.Open(cleanFullPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", getContentType(cleanFullPath))
		http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
	})
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Finnegan",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: createAssetHandler(app),
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{ Appearance: "NSAppearanceNameVibrantLight",
			TitleBar:             mac.TitleBarHiddenInset(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Finnegan",
				Message: "Finnegan Svelte UI (Scattering & Focus)",
			},
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
