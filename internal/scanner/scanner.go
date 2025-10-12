package scanner

import (
	"crypto/sha256"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"photodatabase/internal/database"
	"photodatabase/internal/models"

	"github.com/rwcarlsen/goexif/exif"
	_ "golang.org/x/image/webp"
)

type Scanner struct {
	db           *database.DB
	thumbnailDir string
	supportedExts map[string]bool
}

func New(db *database.DB, thumbnailDir string) *Scanner {
	supportedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".tiff": true,
		".tif":  true,
		".bmp":  true,
		".raw":  true,
		".cr2":  true,
		".nef":  true,
		".arw":  true,
		".dng":  true,
	}

	return &Scanner{
		db:            db,
		thumbnailDir:  thumbnailDir,
		supportedExts: supportedExts,
	}
}

func (s *Scanner) ScanDirectory(rootPath string) error {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", absPath)
	}

	log.Printf("Starting scan of directory: %s", absPath)

	// Create or get directory record
	dirName := filepath.Base(absPath)
	dir, err := s.db.GetDirectory(absPath)
	if err != nil {
		return fmt.Errorf("failed to get directory: %w", err)
	}

	if dir == nil {
		dir, err = s.db.CreateDirectory(absPath, dirName)
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		log.Printf("Created directory record: %s", absPath)
	}

	// Walk through directory
	fileCount := 0
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error walking path %s: %v", path, err)
			return nil // Continue walking
		}

		if info.IsDir() {
			return nil
		}

		if s.isImageFile(path) {
			if err := s.processFile(path, dir.ID, absPath); err != nil {
				log.Printf("Error processing file %s: %v", path, err)
			} else {
				fileCount++
				if fileCount%100 == 0 {
					log.Printf("Processed %d files...", fileCount)
				}
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Update directory last scanned time
	if err := s.db.UpdateDirectoryLastScanned(dir.ID); err != nil {
		log.Printf("Failed to update directory last scanned: %v", err)
	}

	log.Printf("Scan completed. Processed %d files in %s", fileCount, absPath)
	return nil
}

func (s *Scanner) processFile(filePath string, directoryID int, rootPath string) error {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Calculate relative path
	relativePath, err := filepath.Rel(rootPath, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	// Check if file already exists in database
	existingFile, err := s.db.GetFileByPath(filePath)
	if err != nil {
		return fmt.Errorf("failed to check existing file: %w", err)
	}

	// If file exists and hasn't been modified, skip
	if existingFile != nil && existingFile.ModifiedAt.Equal(info.ModTime()) {
		return nil
	}

	// Calculate file hash
	hash, err := s.calculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Get MIME type
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Get image dimensions
	width, height := s.getImageDimensions(filePath)

	// Extract EXIF data
	exifData := s.extractExifData(filePath)

	// Create file record
	file := &models.File{
		DirectoryID:     directoryID,
		Filename:        filepath.Base(filePath),
		RelativePath:    relativePath,
		AbsolutePath:    filePath,
		FileSize:        info.Size(),
		ModifiedAt:      info.ModTime(),
		Hash:            hash,
		MimeType:        mimeType,
		Width:           width,
		Height:          height,
		HasThumbnail:    false,
		ExifDateTime:    exifData.DateTime,
		ExifMake:        exifData.Make,
		ExifModel:       exifData.Model,
		ExifOrientation: exifData.Orientation,
		ExifISO:         exifData.ISO,
		ExifFNumber:     exifData.FNumber,
		ExifExposure:    exifData.Exposure,
		ExifFocalLength: exifData.FocalLength,
		ExifLens:        exifData.Lens,
	}

	if existingFile != nil {
		file.ID = existingFile.ID
		file.CreatedAt = existingFile.CreatedAt
		err = s.db.UpdateFile(file)
	} else {
		err = s.db.CreateFile(file)
	}

	if err != nil {
		return fmt.Errorf("failed to save file to database: %w", err)
	}

	return nil
}

func (s *Scanner) isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return s.supportedExts[ext]
}

func (s *Scanner) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (s *Scanner) getImageDimensions(filePath string) (int, int) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0
	}

	return config.Width, config.Height
}

type ExifData struct {
	DateTime    *time.Time
	Make        string
	Model       string
	Orientation int
	ISO         int
	FNumber     float64
	Exposure    string
	FocalLength float64
	Lens        string
}

func (s *Scanner) extractExifData(filePath string) *ExifData {
	data := &ExifData{
		Orientation: 1, // Default orientation
	}

	file, err := os.Open(filePath)
	if err != nil {
		return data
	}
	defer file.Close()

	x, err := exif.Decode(file)
	if err != nil {
		return data // Return default data if EXIF parsing fails
	}

	// Extract DateTime
	if dateTimeTag, err := x.Get(exif.DateTime); err == nil {
		if dateTimeStr, err := dateTimeTag.StringVal(); err == nil {
			if dt, err := time.Parse("2006:01:02 15:04:05", dateTimeStr); err == nil {
				data.DateTime = &dt
			}
		}
	} else if dateTimeOrigTag, err := x.Get(exif.DateTimeOriginal); err == nil {
		if dateTimeStr, err := dateTimeOrigTag.StringVal(); err == nil {
			if dt, err := time.Parse("2006:01:02 15:04:05", dateTimeStr); err == nil {
				data.DateTime = &dt
			}
		}
	}

	// Extract Make
	if makeTag, err := x.Get(exif.Make); err == nil {
		if makeStr, err := makeTag.StringVal(); err == nil {
			data.Make = strings.TrimSpace(makeStr)
		}
	}

	// Extract Model
	if modelTag, err := x.Get(exif.Model); err == nil {
		if modelStr, err := modelTag.StringVal(); err == nil {
			data.Model = strings.TrimSpace(modelStr)
		}
	}

	// Extract Orientation
	if orientationTag, err := x.Get(exif.Orientation); err == nil {
		if orientationInt, err := orientationTag.Int(0); err == nil {
			data.Orientation = orientationInt
		}
	}

	// Extract ISO
	if isoTag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if isoInt, err := isoTag.Int(0); err == nil {
			data.ISO = isoInt
		}
	}

	// Extract F-Number
	if fNumberTag, err := x.Get(exif.FNumber); err == nil {
		if fNumberStr, err := fNumberTag.StringVal(); err == nil {
			// Simple parsing - most f-numbers are in the format "f/2.8"
			if len(fNumberStr) > 2 && fNumberStr[:2] == "f/" {
				if val, parseErr := strconv.ParseFloat(fNumberStr[2:], 64); parseErr == nil {
					data.FNumber = val
				}
			}
		}
	}

	// Extract Exposure Time
	if exposureTag, err := x.Get(exif.ExposureTime); err == nil {
		if exposureStr, err := exposureTag.StringVal(); err == nil {
			data.Exposure = exposureStr
		}
	}

	// Extract Focal Length
	if focalLengthTag, err := x.Get(exif.FocalLength); err == nil {
		if focalLengthStr, err := focalLengthTag.StringVal(); err == nil {
			if val, parseErr := strconv.ParseFloat(strings.TrimSpace(focalLengthStr), 64); parseErr == nil {
				data.FocalLength = val
			}
		}
	}

	// Extract Lens Model (may not be available in all EXIF data)
	if lensTag, err := x.Get(exif.LensModel); err == nil {
		if lensStr, err := lensTag.StringVal(); err == nil {
			data.Lens = strings.TrimSpace(lensStr)
		}
	}

	return data
}

func (s *Scanner) RescanAll() error {
	directories, err := s.db.GetAllDirectories()
	if err != nil {
		return fmt.Errorf("failed to get directories: %w", err)
	}

	for _, dir := range directories {
		log.Printf("Rescanning directory: %s", dir.Path)
		if err := s.ScanDirectory(dir.Path); err != nil {
			log.Printf("Failed to rescan directory %s: %v", dir.Path, err)
			continue
		}
	}

	// Clean up orphaned files
	log.Println("Cleaning up orphaned files...")
	if err := s.db.DeleteOrphanedFiles(); err != nil {
		log.Printf("Failed to clean up orphaned files: %v", err)
	}

	log.Println("Rescan completed")
	return nil
}