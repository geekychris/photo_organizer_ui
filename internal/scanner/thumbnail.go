package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

// GenerateThumbnail creates a thumbnail for the given image file
func (s *Scanner) GenerateThumbnail(filePath string, size int) (string, error) {
	// Create thumbnail filename
	hash, err := s.calculateFileHash(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate hash for thumbnail: %w", err)
	}
	
	thumbnailName := fmt.Sprintf("%s_%d.jpg", hash, size)
	thumbnailPath := filepath.Join(s.thumbnailDir, thumbnailName)
	
	// Check if thumbnail already exists
	if _, err := os.Stat(thumbnailPath); err == nil {
		return thumbnailPath, nil
	}
	
	// First, try with the Go imaging library
	err = s.generateThumbnailWithImaging(filePath, thumbnailPath, size)
	if err == nil {
		return thumbnailPath, nil
	}
	
	// If that fails, try with ImageMagick as fallback
	err = s.generateThumbnailWithImageMagick(filePath, thumbnailPath, size)
	if err != nil {
		return "", fmt.Errorf("failed to generate thumbnail with both methods: %w", err)
	}
	
	return thumbnailPath, nil
}

// generateThumbnailWithImaging uses the Go imaging library
func (s *Scanner) generateThumbnailWithImaging(filePath, thumbnailPath string, size int) error {
	// Skip problematic TIFF files that are known to cause issues
	if s.isProblematicTIFF(filePath) {
		return fmt.Errorf("skipping problematic TIFF file")
	}
	
	// Open source image
	src, err := imaging.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	
	// Create thumbnail
	thumbnail := imaging.Fit(src, size, size, imaging.Lanczos)
	
	// Save thumbnail as JPEG
	err = imaging.Save(thumbnail, thumbnailPath, imaging.JPEGQuality(85))
	if err != nil {
		return fmt.Errorf("failed to save thumbnail: %w", err)
	}
	
	return nil
}

// generateThumbnailWithImageMagick uses ImageMagick as fallback
func (s *Scanner) generateThumbnailWithImageMagick(filePath, thumbnailPath string, size int) error {
	// Check if ImageMagick is available
	if !s.isImageMagickAvailable() {
		return fmt.Errorf("ImageMagick not available")
	}
	
	// Use magick command to create thumbnail
	cmd := exec.Command("magick", filePath, "-resize", fmt.Sprintf("%dx%d", size, size), "-quality", "85", thumbnailPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ImageMagick conversion failed: %w, output: %s", err, string(output))
	}
	
	return nil
}

// isProblematicTIFF checks if a TIFF file is likely to cause issues with the Go imaging library
func (s *Scanner) isProblematicTIFF(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".tiff" && ext != ".tif" {
		return false
	}
	
	// Check file for known problematic patterns
	file, err := os.Open(filePath)
	if err != nil {
		return true // If we can't open it, consider it problematic
	}
	defer file.Close()
	
	// Read first 1KB to check TIFF header and compression
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil || n < 8 {
		return true // File too small or unreadable
	}
	
	// Check TIFF magic number
	if !((buffer[0] == 0x49 && buffer[1] == 0x49 && buffer[2] == 0x2A && buffer[3] == 0x00) || // Little-endian
		(buffer[0] == 0x4D && buffer[1] == 0x4D && buffer[2] == 0x00 && buffer[3] == 0x2A)) { // Big-endian
		return true // Invalid TIFF header
	}
	
	// Check file size - if too small, likely corrupted
	fileInfo, err := file.Stat()
	if err != nil {
		return true
	}
	
	if fileInfo.Size() < 1000 {
		return true // File too small for a valid TIFF
	}
	
	// Check for specific problematic files by name patterns that we've seen fail
	filename := filepath.Base(filePath)
	problematicPatterns := []string{
		"Slides_01_", // These files showed "unexpected EOF" errors
		"Amiga cpu w monitor_", // This file showed "unexpected EOF" errors
	}
	
	for _, pattern := range problematicPatterns {
		if strings.Contains(filename, pattern) {
			return true
		}
	}
	
	// Try to detect LZW compression by looking at the TIFF directory structure
	// This is a simplified check - a full TIFF parser would be more accurate
	return s.hasProblematicCompression(buffer, fileInfo.Size())
}

// hasProblematicCompression checks if the TIFF uses compression that Go imaging library struggles with
func (s *Scanner) hasProblematicCompression(buffer []byte, fileSize int64) bool {
	// This is a simplified heuristic. A proper implementation would parse TIFF IFD entries
	// to find the compression tag (259) and check its value
	
	// Common indicators of problematic files:
	// 1. Files that are much smaller than expected for their dimensions
	// 2. Specific byte patterns that indicate LZW or other problematic compression
	
	// If file is suspiciously small relative to what we'd expect for uncompressed data,
	// it's likely using heavy compression that might be problematic
	if fileSize < 1000000 && len(buffer) > 100 {
		// Look for LZW compression indicators in the data stream
		// LZW compressed data often has specific patterns
		for i := 0; i < len(buffer)-4; i++ {
			// Look for compression tag 259 (0x0103) with value 5 (LZW)
			if buffer[i] == 0x03 && buffer[i+1] == 0x01 && i+8 < len(buffer) {
				// Check if the value at this tag position is 5 (LZW)
				if buffer[i+6] == 0x05 && buffer[i+7] == 0x00 {
					return true // LZW compression detected
				}
			}
		}
	}
	
	return false
}

// isImageMagickAvailable checks if ImageMagick is available on the system
func (s *Scanner) isImageMagickAvailable() bool {
	_, err := exec.LookPath("magick")
	if err == nil {
		return true
	}
	
	// Try older convert command as fallback
	_, err = exec.LookPath("convert")
	return err == nil
}

// GenerateThumbnails creates thumbnails for all files that don't have them
func (s *Scanner) GenerateThumbnails(size int) error {
	// This would be called as a separate process
	// For now, we'll generate thumbnails on-demand in the web server
	return nil
}