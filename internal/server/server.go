package server

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"photodatabase/internal/config"
	"photodatabase/internal/database"
	"photodatabase/internal/models"
	"photodatabase/internal/scanner"

	"github.com/gorilla/mux"
)

type Server struct {
	db       *database.DB
	config   *config.Config
	scanner  *scanner.Scanner
	router   *mux.Router
	templates map[string]*template.Template
}

func New(db *database.DB, cfg *config.Config) *Server {
	s := &Server{
		db:     db,
		config: cfg,
		scanner: scanner.New(db, cfg.ThumbnailDir),
		router: mux.NewRouter(),
		templates: make(map[string]*template.Template),
	}

	s.setupRoutes()
	s.loadTemplates()
	return s
}

func (s *Server) setupRoutes() {
	// Static files
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))
	s.router.PathPrefix("/thumbnails/").Handler(http.StripPrefix("/thumbnails/", http.FileServer(http.Dir(s.config.ThumbnailDir))))
	
	// Image serving endpoint
	s.router.HandleFunc("/image/{id:[0-9]+}", s.serveImage).Methods("GET")
	s.router.HandleFunc("/thumbnail/{id:[0-9]+}", s.serveThumbnail).Methods("GET")

	// Web pages
	s.router.HandleFunc("/", s.indexHandler).Methods("GET")
	s.router.HandleFunc("/browse", s.browseHandler).Methods("GET")
	s.router.HandleFunc("/duplicates", s.duplicatesHandler).Methods("GET")
	s.router.HandleFunc("/directories", s.directoriesHandler).Methods("GET")

	// API endpoints
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/stats", s.apiStats).Methods("GET")
	api.HandleFunc("/files", s.apiFiles).Methods("GET")
	api.HandleFunc("/files/grouped", s.apiFilesGrouped).Methods("GET")
	api.HandleFunc("/files/{id:[0-9]+}", s.apiFile).Methods("GET")
	api.HandleFunc("/duplicates", s.apiDuplicates).Methods("GET")
	api.HandleFunc("/duplicates/{hash}", s.apiDuplicateFiles).Methods("GET")
	api.HandleFunc("/directories", s.apiDirectories).Methods("GET")
	api.HandleFunc("/directories/tree", s.apiDirectoryTree).Methods("GET")
	api.HandleFunc("/dates/tree", s.apiDateTree).Methods("GET")
	api.HandleFunc("/scan", s.apiScan).Methods("POST")
	api.HandleFunc("/rescan", s.apiRescan).Methods("POST")
}

func (s *Server) loadTemplates() {
	templateDir := "web/templates"
	
	// Load templates
	s.templates["index"] = template.Must(template.ParseFiles(
		filepath.Join(templateDir, "base.html"),
		filepath.Join(templateDir, "index.html"),
	))
	
	s.templates["browse"] = template.Must(template.ParseFiles(
		filepath.Join(templateDir, "base.html"),
		filepath.Join(templateDir, "browse.html"),
	))
	
	s.templates["duplicates"] = template.Must(template.ParseFiles(
		filepath.Join(templateDir, "base.html"),
		filepath.Join(templateDir, "duplicates.html"),
	))
	
	s.templates["directories"] = template.Must(template.ParseFiles(
		filepath.Join(templateDir, "base.html"),
		filepath.Join(templateDir, "directories.html"),
	))
}

func (s *Server) Start(port string) error {
	log.Printf("Server starting on http://localhost:%s", port)
	return http.ListenAndServe(":"+port, s.router)
}

// Web handlers

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetStats()
	if err != nil {
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	data := struct {
		Stats *models.Stats
	}{
		Stats: stats,
	}

	if err := s.templates["index"].Execute(w, data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (s *Server) browseHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.templates["browse"].Execute(w, nil); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (s *Server) duplicatesHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.templates["duplicates"].Execute(w, nil); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (s *Server) directoriesHandler(w http.ResponseWriter, r *http.Request) {
	directories, err := s.db.GetAllDirectories()
	if err != nil {
		http.Error(w, "Failed to get directories", http.StatusInternalServerError)
		return
	}

	data := struct {
		Directories []models.Directory
	}{
		Directories: directories,
	}

	if err := s.templates["directories"].Execute(w, data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// API handlers

func (s *Server) apiStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetStats()
	if err != nil {
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) apiFiles(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	params := &models.BrowseParams{
		Limit:  50,
		Offset: 0,
		OrderBy: "path",
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 200 {
			params.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = offset
		}
	}

	if orderBy := r.URL.Query().Get("order_by"); orderBy != "" {
		params.OrderBy = orderBy
	}

	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			params.Year = year
		}
	}

	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		if month, err := strconv.Atoi(monthStr); err == nil && month >= 1 && month <= 12 {
			params.Month = month
		}
	}

	if dayStr := r.URL.Query().Get("day"); dayStr != "" {
		if day, err := strconv.Atoi(dayStr); err == nil && day >= 1 && day <= 31 {
			params.Day = day
		}
	}

	params.Directory = r.URL.Query().Get("directory")
	params.Search = r.URL.Query().Get("search")
	params.GroupBy = r.URL.Query().Get("group_by")

	// Parse date range
	if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			params.DateFrom = &dateFrom
		}
	}

	if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
			// Set to end of day
			dateToEnd := dateTo.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			params.DateTo = &dateToEnd
		}
	}

	result, err := s.db.BrowseFiles(params)
	if err != nil {
		http.Error(w, "Failed to browse files", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) apiFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	file, err := s.db.GetFileByID(id)
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusInternalServerError)
		return
	}

	if file == nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}

func (s *Server) apiDuplicates(w http.ResponseWriter, r *http.Request) {
	duplicates, err := s.db.GetDuplicates()
	if err != nil {
		http.Error(w, "Failed to get duplicates", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(duplicates)
}

func (s *Server) apiDuplicateFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	files, err := s.db.GetFilesByHash(hash)
	if err != nil {
		http.Error(w, "Failed to get files", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (s *Server) apiDirectories(w http.ResponseWriter, r *http.Request) {
	directories, err := s.db.GetAllDirectories()
	if err != nil {
		http.Error(w, "Failed to get directories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(directories)
}

func (s *Server) apiScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Directory string `json:"directory"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Directory == "" {
		http.Error(w, "Directory is required", http.StatusBadRequest)
		return
	}

	// Run scan in background
	go func() {
		log.Printf("Starting background scan of: %s", req.Directory)
		if err := s.scanner.ScanDirectory(req.Directory); err != nil {
			log.Printf("Scan failed: %v", err)
		} else {
			log.Printf("Scan completed: %s", req.Directory)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "scan started"})
}

func (s *Server) apiFilesGrouped(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters (same as apiFiles)
	params := &models.BrowseParams{
		Limit:  50,
		Offset: 0,
		OrderBy: "path",
		GroupBy: "directory", // Default to directory grouping
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 200 {
			params.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			params.Offset = offset
		}
	}

	if orderBy := r.URL.Query().Get("order_by"); orderBy != "" {
		params.OrderBy = orderBy
	}

	if groupBy := r.URL.Query().Get("group_by"); groupBy != "" {
		params.GroupBy = groupBy
	}

	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			params.Year = year
		}
	}

	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		if month, err := strconv.Atoi(monthStr); err == nil && month >= 1 && month <= 12 {
			params.Month = month
		}
	}

	if dayStr := r.URL.Query().Get("day"); dayStr != "" {
		if day, err := strconv.Atoi(dayStr); err == nil && day >= 1 && day <= 31 {
			params.Day = day
		}
	}

	params.Directory = r.URL.Query().Get("directory")
	params.Search = r.URL.Query().Get("search")

	// Parse date range
	if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			params.DateFrom = &dateFrom
		}
	}

	if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
			// Set to end of day
			dateToEnd := dateTo.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			params.DateTo = &dateToEnd
		}
	}

	result, err := s.db.BrowseFilesGrouped(params)
	if err != nil {
		http.Error(w, "Failed to browse files", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) apiDirectoryTree(w http.ResponseWriter, r *http.Request) {
	tree, err := s.db.GetDirectoryTree()
	if err != nil {
		http.Error(w, "Failed to get directory tree", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tree)
}

func (s *Server) apiDateTree(w http.ResponseWriter, r *http.Request) {
	tree, err := s.db.GetDateTree()
	if err != nil {
		http.Error(w, "Failed to get date tree", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tree)
}

func (s *Server) apiRescan(w http.ResponseWriter, r *http.Request) {
	// Run rescan in background
	go func() {
		log.Println("Starting background rescan")
		if err := s.scanner.RescanAll(); err != nil {
			log.Printf("Rescan failed: %v", err)
		} else {
			log.Println("Rescan completed")
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rescan started"})
}

// File serving handlers

func (s *Server) serveImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	file, err := s.db.GetFileByID(id)
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusInternalServerError)
		return
	}

	if file == nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Check if file exists on disk
	if _, err := os.Stat(file.AbsolutePath); os.IsNotExist(err) {
		http.Error(w, "File not found on disk", http.StatusNotFound)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Cache-Control", "public, max-age=86400")

	// Serve the file
	http.ServeFile(w, r, file.AbsolutePath)
}

func (s *Server) serveThumbnail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	file, err := s.db.GetFileByID(id)
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusInternalServerError)
		return
	}

	if file == nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Generate thumbnail if it doesn't exist
	if !file.HasThumbnail || file.ThumbnailPath == "" {
		thumbnailPath, err := s.scanner.GenerateThumbnail(file.AbsolutePath, s.config.ThumbnailSize)
		if err != nil {
			log.Printf("Failed to generate thumbnail for %s: %v", file.AbsolutePath, err)
			http.Error(w, "Failed to generate thumbnail", http.StatusInternalServerError)
			return
		}

		// Update file record with thumbnail info
		file.HasThumbnail = true
		file.ThumbnailPath = thumbnailPath
		if err := s.db.UpdateFile(file); err != nil {
			log.Printf("Failed to update file with thumbnail info: %v", err)
		}
	}

	// Check if thumbnail exists on disk
	if _, err := os.Stat(file.ThumbnailPath); os.IsNotExist(err) {
		// Try to regenerate thumbnail
		thumbnailPath, err := s.scanner.GenerateThumbnail(file.AbsolutePath, s.config.ThumbnailSize)
		if err != nil {
			log.Printf("Failed to regenerate thumbnail for %s: %v", file.AbsolutePath, err)
			http.Error(w, "Thumbnail not available", http.StatusNotFound)
			return
		}
		file.ThumbnailPath = thumbnailPath
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400")

	// Serve the thumbnail
	http.ServeFile(w, r, file.ThumbnailPath)
}
