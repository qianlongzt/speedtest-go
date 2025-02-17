package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"

	"github.com/librespeed/speedtest/config"
	"github.com/librespeed/speedtest/results"
)

const (
	// chunk size is 1 mib
	chunkSize = 1048576
)

//go:embed assets
var defaultAssets embed.FS

var (
	// generate random data for download test on start to minimize runtime overhead
	randomData = getRandomData(chunkSize)
)

func ListenAndServe(ctx context.Context, conf *config.Config) error {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.GetHead)

	cs := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS", "HEAD"},
		AllowedHeaders: []string{"*"},
	})

	r.Use(cs.Handler)
	r.Use(middleware.NoCache)
	r.Use(middleware.Recoverer)

	var assetFS http.FileSystem
	if fi, err := os.Stat(conf.AssetsPath); os.IsNotExist(err) || !fi.IsDir() {
		slog.Warn("Configured asset path does not exist or is not a directory, using default assets",
			slog.Any("path", conf.AssetsPath))
		sub, err := fs.Sub(defaultAssets, "assets")
		if err != nil {
			return fmt.Errorf("failed when processing default assets: %w", err)
		}
		assetFS = http.FS(sub)
	} else {
		assetFS = justFilesFilesystem{fs: http.Dir(conf.AssetsPath), readDirBatchSize: 2}
	}
	base := conf.BaseURL
	if base == "" {
		base = "/"
	}
	r.Route(base, func(r chi.Router) {
		r.Get("/*", pages(assetFS, conf.BaseURL))
		r.HandleFunc("/empty", empty)
		r.HandleFunc("/backend/empty", empty)
		r.Get("/garbage", garbage)
		r.Get("/backend/garbage", garbage)
		r.Get("/getIP", getIP)
		r.Get("/backend/getIP", getIP)
		r.Get("/results", results.DrawPNG)
		r.Get("/results/", results.DrawPNG)
		r.Get("/backend/results", results.DrawPNG)
		r.Get("/backend/results/", results.DrawPNG)
		r.Post("/results/telemetry", results.Record)
		r.Post("/backend/results/telemetry", results.Record)
		r.HandleFunc("/stats", results.Stats)
		r.HandleFunc("/backend/stats", results.Stats)

		// PHP frontend default values compatibility
		r.HandleFunc("/empty.php", empty)
		r.HandleFunc("/backend/empty.php", empty)
		r.Get("/garbage.php", garbage)
		r.Get("/backend/garbage.php", garbage)
		r.Get("/getIP.php", getIP)
		r.Get("/backend/getIP.php", getIP)
		r.Post("/results/telemetry.php", results.Record)
		r.Post("/backend/results/telemetry.php", results.Record)
		r.HandleFunc("/stats.php", results.Stats)
		r.HandleFunc("/backend/stats.php", results.Stats)
	})

	return startListener(ctx, conf, r)
}

func trimPrefix(url string, base string) string {
	return strings.TrimPrefix(url, base+"/")
}
func pages(fs http.FileSystem, BaseURL string) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if BaseURL != "" {
			r.URL.Path = trimPrefix(r.URL.Path, BaseURL)
		}
		if r.RequestURI == "/" {
			r.RequestURI = "/index.html"
		}

		http.FileServer(fs).ServeHTTP(w, r)
	}

	return fn
}

func empty(w http.ResponseWriter, r *http.Request) {
	_, err := io.Copy(io.Discard, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
}

func garbage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Description", "File Transfer")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=random.dat")
	w.Header().Set("Content-Transfer-Encoding", "binary")

	// chunk size set to 4 by default
	chunks := 4

	ckSize := r.FormValue("ckSize")
	if ckSize != "" {
		i, err := strconv.ParseInt(ckSize, 10, 64)
		if err != nil {
			slog.Error("Invalid chunk size: %s", slog.Any("ckSize", ckSize))
			slog.Warn("Will use default value %d", slog.Any("ckSize", chunks))
		} else {
			// limit max chunk size to 1024
			if i > 1024 {
				chunks = 1024
			} else {
				chunks = int(i)
			}
		}
	}

	for i := 0; i < chunks; i++ {
		if _, err := w.Write(randomData); err != nil {
			slog.Error("Error writing back to client",
				slog.Any("chunk number", i),
				slog.Any("error", err),
			)

			break
		}
	}
}

func getIP(w http.ResponseWriter, r *http.Request) {
	var ret results.Result

	clientIP := r.RemoteAddr
	clientIP = strings.ReplaceAll(clientIP, "::ffff:", "")

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		clientIP = ip
	}

	isSpecialIP := true
	switch {
	case clientIP == "::1":
		ret.ProcessedString = clientIP + " - localhost IPv6 access"
	case strings.HasPrefix(clientIP, "fe80:"):
		ret.ProcessedString = clientIP + " - link-local IPv6 access"
	case strings.HasPrefix(clientIP, "127."):
		ret.ProcessedString = clientIP + " - localhost IPv4 access"
	case strings.HasPrefix(clientIP, "10."):
		ret.ProcessedString = clientIP + " - private IPv4 access"
	case regexp.MustCompile(`^172\.(1[6-9]|2\d|3[01])\.`).MatchString(clientIP):
		ret.ProcessedString = clientIP + " - private IPv4 access"
	case strings.HasPrefix(clientIP, "192.168"):
		ret.ProcessedString = clientIP + " - private IPv4 access"
	case strings.HasPrefix(clientIP, "169.254"):
		ret.ProcessedString = clientIP + " - link-local IPv4 access"
	case regexp.MustCompile(`^100\.([6-9][0-9]|1[0-2][0-7])\.`).MatchString(clientIP):
		ret.ProcessedString = clientIP + " - CGNAT IPv4 access"
	default:
		isSpecialIP = false
	}

	if isSpecialIP {
		b, _ := json.Marshal(&ret)
		if _, err := w.Write(b); err != nil {
			slog.Error("writing to client", slog.Any("error", err))
		}
		return
	}

	getISPInfo := r.FormValue("isp") == "true"
	distanceUnit := r.FormValue("distance")

	ret.ProcessedString = clientIP

	if getISPInfo {
		ispInfo := getIPInfo(clientIP)
		ret.RawISPInfo = ispInfo

		removeRegexp := regexp.MustCompile(`AS\d+\s`)
		isp := removeRegexp.ReplaceAllString(ispInfo.Organization, "")

		if isp == "" {
			isp = "Unknown ISP"
		}

		if ispInfo.Country != "" {
			isp += ", " + ispInfo.Country
		}

		if ispInfo.Location != "" {
			isp += " (" + calculateDistance(ispInfo.Location, distanceUnit) + ")"
		}

		ret.ProcessedString += " - " + isp
	}

	render.JSON(w, r, ret)
}
