package api

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jaym/clyper/metadata"
	processor "github.com/jaym/clyper/processors"
)

type ApiHandler struct {
	db           *metadata.Database
	gifOptions   processor.GifOptions
	objstorePath string
}

func NewApiHandler(db *metadata.Database, objstorePath string, gifOptions processor.GifOptions) http.Handler {
	mux := http.NewServeMux()

	apiHandler := &ApiHandler{
		db:           db,
		objstorePath: objstorePath,
		gifOptions:   gifOptions,
	}

	mux.HandleFunc("/search", apiHandler.searchHandler)
	mux.HandleFunc("/thumbs/{season}/{episode}/{timestamp}", apiHandler.thumbsHandler)
	mux.HandleFunc("/thumb/{season}/{episode}/{timestamp}", apiHandler.thumbHandler)
	mux.HandleFunc("/gif/{season}/{episode}/{start}/{end}", apiHandler.gifHandler)

	return allowCORS(mux)
}

func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	})
}

func (h *ApiHandler) searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	// Handle the search logic
	results, err := h.db.Search(r.Context(), query)
	if err != nil {
		http.Error(w, "Failed to search", http.StatusInternalServerError)
		return
	}
	if len(results) == 0 {
		results = []metadata.SearchResult{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

type ThumbnailItem struct {
	Timestamp int `json:"timestamp"`
}

func (h *ApiHandler) thumbHandler(w http.ResponseWriter, r *http.Request) {
	seasonStr := r.PathValue("season")
	episodeStr := r.PathValue("episode")
	timestampStr := r.PathValue("timestamp")
	// strip extension from timestampStr
	timestampStr, _, _ = strings.Cut(timestampStr, ".")

	season, err := strconv.Atoi(seasonStr)
	if err != nil {
		http.Error(w, "Invalid season", http.StatusBadRequest)
		return
	}
	episode, err := strconv.Atoi(episodeStr)
	if err != nil {
		http.Error(w, "Invalid episode", http.StatusBadRequest)
		return
	}
	timestamp, err := strconv.Atoi(timestampStr)
	if err != nil {
		http.Error(w, "Invalid timestamp", http.StatusBadRequest)
		return
	}

	thumbs, err := h.db.ListThumbnails(r.Context(), season, episode, timestamp, 1, false)
	if err != nil {
		http.Error(w, "Failed to list thumbnails", http.StatusInternalServerError)
		return
	}

	if len(thumbs) == 0 {
		http.Error(w, "Thumbnail not found", http.StatusNotFound)
		return
	}

	thumb := thumbs[0]

	// Read the thumbnail object
	thumbPath := path.Join(h.objstorePath, thumb.Key)
	obj, err := os.Open(thumbPath)
	if err != nil {
		http.Error(w, "Failed to read thumbnail", http.StatusInternalServerError)
		return
	}
	defer obj.Close()

	// Serve the thumbnail (always a JPEG)
	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, obj)
}

func (h *ApiHandler) thumbsHandler(w http.ResponseWriter, r *http.Request) {
	reverseStr := r.URL.Query().Get("reverse")
	seasonStr := r.PathValue("season")
	episodeStr := r.PathValue("episode")
	timestampStr := r.PathValue("timestamp")
	// strip extension from timestampStr
	timestampStr, _, _ = strings.Cut(timestampStr, ".")

	season, err := strconv.Atoi(seasonStr)
	if err != nil {
		http.Error(w, "Invalid season", http.StatusBadRequest)
		return
	}
	episode, err := strconv.Atoi(episodeStr)
	if err != nil {
		http.Error(w, "Invalid episode", http.StatusBadRequest)
		return
	}
	timestamp, err := strconv.Atoi(timestampStr)
	if err != nil {
		http.Error(w, "Invalid timestamp", http.StatusBadRequest)
		return
	}
	var reverse bool
	if r.URL.Query().Has("reverse") {
		var err error
		reverse, err = strconv.ParseBool(reverseStr)
		if err != nil {
			http.Error(w, "Invalid reverse", http.StatusBadRequest)
			return
		}
	}

	thumbs, err := h.db.ListThumbnails(r.Context(), season, episode, timestamp, 25, reverse)
	if err != nil {
		http.Error(w, "Failed to list thumbnails", http.StatusInternalServerError)
		return
	}

	items := make([]ThumbnailItem, 0, len(thumbs))
	for _, thumb := range thumbs {
		items = append(items, ThumbnailItem{
			Timestamp: thumb.Start,
		})
	}

	// Handle the caption logic
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *ApiHandler) gifHandler(w http.ResponseWriter, r *http.Request) {
	seasonStr := r.PathValue("season")
	episodeStr := r.PathValue("episode")
	startStr := r.PathValue("start")
	endStr := r.PathValue("end")
	// strip extension from endStr
	endStr, _, _ = strings.Cut(endStr, ".")

	b64Lines := r.URL.Query().Get("b64lines")
	text := r.URL.Query().Get("text")

	season, err := strconv.Atoi(seasonStr)
	if err != nil {
		http.Error(w, "Invalid season", http.StatusBadRequest)
		return
	}
	episode, err := strconv.Atoi(episodeStr)
	if err != nil {
		http.Error(w, "Invalid episode", http.StatusBadRequest)
		return
	}
	start, err := strconv.Atoi(startStr)
	if err != nil {
		http.Error(w, "Invalid start", http.StatusBadRequest)
		return
	}
	end, err := strconv.Atoi(endStr)
	if err != nil {
		http.Error(w, "Invalid end", http.StatusBadRequest)
		return
	}

	if end < start {
		http.Error(w, "Invalid time range", http.StatusBadRequest)
		return
	}

	if end-start > processor.MaxGifDurationMS {
		http.Error(w, "Invalid time range", http.StatusBadRequest)
		return
	}

	var captionLines string
	if b64Lines != "" {
		captionBytes, err := base64.StdEncoding.DecodeString(b64Lines)
		if err != nil {
			http.Error(w, "Invalid caption", http.StatusBadRequest)
			return
		}
		captionLines = string(captionBytes)
	} else {
		captionLines = text
	}

	videoFileKey, err := h.db.GetVideoFileKey(r.Context(), season, episode)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
	}
	videoFilePath := path.Join(h.objstorePath, videoFileKey)

	// Handle the GIF logic
	outputDir, err := os.MkdirTemp("", "clyper")
	if err != nil {
		http.Error(w, "Failed to create temp dir", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(outputDir)

	outputFile := path.Join(outputDir, "output.gif")
	opts := h.gifOptions
	opts.Text = captionLines

	err = processor.MakeGif(videoFilePath, outputFile, start, end, opts)
	if err != nil {
		http.Error(w, "Failed to create gif", http.StatusInternalServerError)
		return
	}

	// Read the GIF
	gif, err := os.Open(outputFile)
	if err != nil {
		http.Error(w, "Failed to read gif", http.StatusInternalServerError)
		return
	}

	// Serve the GIF
	w.Header().Set("Content-Type", "image/gif")
	// Cache the gif for 1 day
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, gif)
}
