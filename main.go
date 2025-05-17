package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// Config represents the application configuration
type Config struct {
	Port              string            `json:"port"`
	UploadDirectory   string            `json:"uploadDirectory"`
	FileTypeOpeners   map[string]string `json:"fileTypeOpeners"`
	DefaultOpener     string            `json:"defaultOpener"`
	MaxUploadSize     int64             `json:"maxUploadSize"`
	AllowedExtensions []string          `json:"allowedExtensions"`
}

// Global configuration
var config Config

func init() {
	// Set default configuration
	config = Config{
		Port:            "8080",
		UploadDirectory: "/tmp/agent-uploads",
		FileTypeOpeners: map[string]string{
			".pdf":  "okular",
			".txt":  "gedit",
			".png":  "eog",
			".jpg":  "eog",
			".jpeg": "eog",
			".webp": "eog",
			".gif":  "eog",
			".mp4":  "vlc",
			".mp3":  "vlc",
			".docx": "desktopeditors",
			".xlsx": "desktopeditors",
			".pptx": "desktopeditors",
			".odt":  "desktopeditors",
			".ods":  "desktopeditors",
			".odp":  "desktopeditors",
			".csv":  "desktopeditors",
			// Add archive file formats with xarchiver
			".zip": "file-roller",
			".tar": "file-roller",
			".gz":  "file-roller",
			".bz2": "file-roller",
			".xz":  "file-roller",
			".7z":  "file-roller",
			".rar": "file-roller",
			// Additional video formats for vlc
			".avi":  "vlc",
			".mkv":  "vlc",
			".mov":  "vlc",
			".wmv":  "vlc",
			".flv":  "vlc",
			".webm": "vlc",
		},
		DefaultOpener: "xdg-open",
		MaxUploadSize: 50 * 1024 * 1024, // 50MB
		AllowedExtensions: []string{".pdf", ".txt", ".png", ".jpg", ".jpeg", ".gif", ".mp4", ".mp3", ".docx", ".xlsx", ".pptx", ".odt", ".ods", ".odp", ".csv",
			".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm"},
	}

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(config.UploadDirectory, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

}


// FileResponse represents the response sent after file upload
type FileResponse struct {
	Success      bool   `json:"success"`
	FilePath     string `json:"filePath,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

func main() {
	// Create a default gin router with Logger and Recovery middleware
	r := gin.Default()

	// Add CORS middleware
	r.Use(corsMiddleware())

	// Routes
	r.GET("/health", healthCheckHandler)
	r.POST("/upload", uploadFileHandler)
	r.GET("/open/:filename", openFileHandler)

	// Start the server
	log.Printf("File Opener Agent starting on port %s...", config.Port)
	log.Printf("Upload directory: %s", config.UploadDirectory)
	log.Fatal(r.Run(":" + config.Port))
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// healthCheckHandler returns a simple health check response
func healthCheckHandler(c *gin.Context) {
	response := map[string]string{
		"status":    "OK",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// uploadFileHandler handles file uploads
func uploadFileHandler(c *gin.Context) {
	// Limit request size
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, config.MaxUploadSize)

	// Get file from request
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		respondWithError(c, "Failed to get file from request: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !contains(config.AllowedExtensions, ext) && len(config.AllowedExtensions) > 0 {
		respondWithError(c, "File type not allowed", http.StatusBadRequest)
		return
	}

	// Generate unique filename to prevent collisions
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	filePath := filepath.Join(config.UploadDirectory, filename)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		respondWithError(c, "Failed to create destination file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy uploaded file to destination
	if _, err := io.Copy(dst, file); err != nil {
		respondWithError(c, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if we should automatically open the file
	openNow := c.PostForm("openNow")
	if openNow == "true" {
		go openFile(filePath)
	}

	// Return success response
	c.JSON(http.StatusOK, FileResponse{
		Success:  true,
		FilePath: filePath,
	})
}

// openFileHandler opens a previously uploaded file
func openFileHandler(c *gin.Context) {
	filename := c.Param("filename")

	// Ensure filename is safe (no directory traversal)
	if strings.Contains(filename, "..") {
		respondWithError(c, "Invalid filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(config.UploadDirectory, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		respondWithError(c, "File not found", http.StatusNotFound)
		return
	}

	// Create a channel to signal when the file open operation is complete
	done := make(chan bool, 1)

	// Run the file handling in a goroutine
	go func() {
		openFile(filePath)
		done <- true
	}()

	// Respond immediately without waiting for the open operation to complete
	c.JSON(http.StatusOK, FileResponse{
		Success:  true,
		FilePath: filePath,
	})

	// Optionally, wait for the operation to complete in the background
	// This is non-blocking for the HTTP response
	go func() {
		<-done
		log.Printf("File opening operation completed for %s", filePath)
	}()
}

// openFile opens a file with the appropriate application based on file type
func openFile(filePath string) {
	ext := strings.ToLower(filepath.Ext(filePath))
	opener, exists := config.FileTypeOpeners[ext]

	if !exists {
		opener = config.DefaultOpener
	}

	// Check if the application exists/is available
	_, err := exec.LookPath(opener)
	if err != nil {
		log.Printf("Warning: %s not found in PATH, falling back to default opener", opener)
		opener = config.DefaultOpener
	}

	// Add command-line options for specific applications
	var cmd *exec.Cmd
	switch opener {
	case "xarchiver":
		// xarchiver needs special handling to properly open archives
		cmd = exec.Command(opener, "--extract-to="+filepath.Dir(filePath), filePath)
	case "vlc":
		// vlc might need some specific options
		cmd = exec.Command(opener, "--no-video-title-show", filePath)
	default:
		cmd = exec.Command(opener, filePath)
	}

	// Start the application
	err = cmd.Start()
	if err != nil {
		log.Printf("Error opening %s with %s: %v", filePath, opener, err)

		// Try fallback to default opener
		if opener != config.DefaultOpener {
			fallbackCmd := exec.Command(config.DefaultOpener, filePath)
			fallbackCmd.Start()
			log.Printf("Attempted fallback to %s", config.DefaultOpener)
		}
	} else {
		log.Printf("Opened %s with %s", filePath, opener)
	}
}

// respondWithError sends an error response
func respondWithError(c *gin.Context, message string, statusCode int) {
	c.JSON(statusCode, FileResponse{
		Success:      false,
		ErrorMessage: message,
	})
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
