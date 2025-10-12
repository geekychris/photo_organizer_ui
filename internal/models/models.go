package models

import (
	"time"
)

type Directory struct {
	ID          int       `json:"id" db:"id"`
	Path        string    `json:"path" db:"path"`
	Name        string    `json:"name" db:"name"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	LastScanned time.Time `json:"last_scanned" db:"last_scanned"`
	FileCount   int       `json:"file_count" db:"file_count"`
}

type File struct {
	ID            int       `json:"id" db:"id"`
	DirectoryID   int       `json:"directory_id" db:"directory_id"`
	Filename      string    `json:"filename" db:"filename"`
	RelativePath  string    `json:"relative_path" db:"relative_path"`
	AbsolutePath  string    `json:"absolute_path" db:"absolute_path"`
	FileSize      int64     `json:"file_size" db:"file_size"`
	ModifiedAt    time.Time `json:"modified_at" db:"modified_at"`
	Hash          string    `json:"hash" db:"hash"`
	MimeType      string    `json:"mime_type" db:"mime_type"`
	Width         int       `json:"width" db:"width"`
	Height        int       `json:"height" db:"height"`
	HasThumbnail  bool      `json:"has_thumbnail" db:"has_thumbnail"`
	ThumbnailPath string    `json:"thumbnail_path" db:"thumbnail_path"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	
	// EXIF data
	ExifDateTime    *time.Time `json:"exif_date_time" db:"exif_date_time"`
	ExifMake        string     `json:"exif_make" db:"exif_make"`
	ExifModel       string     `json:"exif_model" db:"exif_model"`
	ExifOrientation int        `json:"exif_orientation" db:"exif_orientation"`
	ExifISO         int        `json:"exif_iso" db:"exif_iso"`
	ExifFNumber     float64    `json:"exif_fnumber" db:"exif_fnumber"`
	ExifExposure    string     `json:"exif_exposure" db:"exif_exposure"`
	ExifFocalLength float64    `json:"exif_focal_length" db:"exif_focal_length"`
	ExifLens        string     `json:"exif_lens" db:"exif_lens"`
}

type FileDuplicate struct {
	Hash       string   `json:"hash" db:"hash"`
	Count      int      `json:"count" db:"count"`
	TotalSize  int64    `json:"total_size" db:"total_size"`
	Files      []File   `json:"files,omitempty"`
}

type BrowseParams struct {
	Limit      int       `json:"limit"`
	Offset     int       `json:"offset"`
	OrderBy    string    `json:"order_by"` // "path", "date", "exif_date", "name"
	Year       int       `json:"year,omitempty"`
	Month      int       `json:"month,omitempty"`
	Day        int       `json:"day,omitempty"`
	DateFrom   *time.Time `json:"date_from,omitempty"`
	DateTo     *time.Time `json:"date_to,omitempty"`
	Directory  string    `json:"directory,omitempty"`
	Search     string    `json:"search,omitempty"`
	GroupBy    string    `json:"group_by,omitempty"` // "directory", "date", "none"
}

type BrowseResult struct {
	Files      []File `json:"files"`
	Total      int    `json:"total"`
	HasMore    bool   `json:"has_more"`
	Offset     int    `json:"offset"`
}

type Stats struct {
	TotalDirectories int   `json:"total_directories"`
	TotalFiles       int   `json:"total_files"`
	TotalSize        int64 `json:"total_size"`
	DuplicateGroups  int   `json:"duplicate_groups"`
	DuplicateFiles   int   `json:"duplicate_files"`
}

type DirectoryNode struct {
	Path        string           `json:"path"`
	Name        string           `json:"name"`
	FileCount   int             `json:"file_count"`
	Children    []*DirectoryNode `json:"children,omitempty"`
	ParentPath  string          `json:"parent_path,omitempty"`
}

type GroupedBrowseResult struct {
	Groups     []FileGroup `json:"groups"`
	Total      int         `json:"total"`
	HasMore    bool        `json:"has_more"`
	Offset     int         `json:"offset"`
}

type FileGroup struct {
	GroupKey   string `json:"group_key"`
	GroupName  string `json:"group_name"`
	Files      []File `json:"files"`
	FileCount  int    `json:"file_count"`
}

type DateTreeNode struct {
	Type      string          `json:"type"`      // "year", "month", "day"
	Value     string          `json:"value"`     // "2024", "01", "15"
	Label     string          `json:"label"`     // "2024", "January", "Jan 15"
	FileCount int             `json:"file_count"`
	Children  []*DateTreeNode `json:"children,omitempty"`
	Parent    *DateTreeNode   `json:"-"` // Don't serialize to avoid circular refs
}
