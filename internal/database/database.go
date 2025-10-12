package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"photodatabase/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS directories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_scanned DATETIME DEFAULT CURRENT_TIMESTAMP,
		file_count INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		directory_id INTEGER NOT NULL,
		filename TEXT NOT NULL,
		relative_path TEXT NOT NULL,
		absolute_path TEXT UNIQUE NOT NULL,
		file_size INTEGER NOT NULL,
		modified_at DATETIME NOT NULL,
		hash TEXT NOT NULL,
		mime_type TEXT NOT NULL,
		width INTEGER DEFAULT 0,
		height INTEGER DEFAULT 0,
		has_thumbnail BOOLEAN DEFAULT FALSE,
		thumbnail_path TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		
		-- EXIF data
		exif_date_time DATETIME,
		exif_make TEXT,
		exif_model TEXT,
		exif_orientation INTEGER DEFAULT 1,
		exif_iso INTEGER DEFAULT 0,
		exif_fnumber REAL DEFAULT 0,
		exif_exposure TEXT,
		exif_focal_length REAL DEFAULT 0,
		exif_lens TEXT,
		
		FOREIGN KEY (directory_id) REFERENCES directories (id) ON DELETE CASCADE
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);
	CREATE INDEX IF NOT EXISTS idx_files_directory_id ON files(directory_id);
	CREATE INDEX IF NOT EXISTS idx_files_exif_date_time ON files(exif_date_time);
	CREATE INDEX IF NOT EXISTS idx_files_modified_at ON files(modified_at);
	CREATE INDEX IF NOT EXISTS idx_files_absolute_path ON files(absolute_path);
	CREATE INDEX IF NOT EXISTS idx_directories_path ON directories(path);

	-- Trigger to update directory file count
	CREATE TRIGGER IF NOT EXISTS update_directory_file_count
	AFTER INSERT ON files
	BEGIN
		UPDATE directories 
		SET file_count = (SELECT COUNT(*) FROM files WHERE directory_id = NEW.directory_id),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = NEW.directory_id;
	END;

	CREATE TRIGGER IF NOT EXISTS update_directory_file_count_delete
	AFTER DELETE ON files
	BEGIN
		UPDATE directories 
		SET file_count = (SELECT COUNT(*) FROM files WHERE directory_id = OLD.directory_id),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = OLD.directory_id;
	END;
	`

	_, err := db.conn.Exec(schema)
	return err
}

func (db *DB) CreateDirectory(path, name string) (*models.Directory, error) {
	now := time.Now()
	query := `
		INSERT INTO directories (path, name, created_at, updated_at, last_scanned)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, path, name, created_at, updated_at, last_scanned, file_count
	`
	
	var dir models.Directory
	err := db.conn.QueryRow(query, path, name, now, now, now).Scan(
		&dir.ID, &dir.Path, &dir.Name, &dir.CreatedAt, &dir.UpdatedAt, &dir.LastScanned, &dir.FileCount,
	)
	
	return &dir, err
}

func (db *DB) GetDirectory(path string) (*models.Directory, error) {
	query := `SELECT id, path, name, created_at, updated_at, last_scanned, file_count FROM directories WHERE path = ?`
	
	var dir models.Directory
	err := db.conn.QueryRow(query, path).Scan(
		&dir.ID, &dir.Path, &dir.Name, &dir.CreatedAt, &dir.UpdatedAt, &dir.LastScanned, &dir.FileCount,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	
	return &dir, err
}

func (db *DB) GetAllDirectories() ([]models.Directory, error) {
	query := `SELECT id, path, name, created_at, updated_at, last_scanned, file_count FROM directories ORDER BY path`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var directories []models.Directory
	for rows.Next() {
		var dir models.Directory
		err := rows.Scan(
			&dir.ID, &dir.Path, &dir.Name, &dir.CreatedAt, &dir.UpdatedAt, &dir.LastScanned, &dir.FileCount,
		)
		if err != nil {
			return nil, err
		}
		directories = append(directories, dir)
	}
	
	return directories, rows.Err()
}

func (db *DB) UpdateDirectoryLastScanned(id int) error {
	query := `UPDATE directories SET last_scanned = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

func (db *DB) CreateFile(file *models.File) error {
	query := `
		INSERT INTO files (
			directory_id, filename, relative_path, absolute_path, file_size, modified_at, 
			hash, mime_type, width, height, has_thumbnail, thumbnail_path,
			exif_date_time, exif_make, exif_model, exif_orientation, exif_iso, 
			exif_fnumber, exif_exposure, exif_focal_length, exif_lens
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at, updated_at
	`
	
	err := db.conn.QueryRow(query,
		file.DirectoryID, file.Filename, file.RelativePath, file.AbsolutePath, file.FileSize,
		file.ModifiedAt, file.Hash, file.MimeType, file.Width, file.Height, file.HasThumbnail,
		file.ThumbnailPath, file.ExifDateTime, file.ExifMake, file.ExifModel, file.ExifOrientation,
		file.ExifISO, file.ExifFNumber, file.ExifExposure, file.ExifFocalLength, file.ExifLens,
	).Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt)
	
	return err
}

func (db *DB) UpdateFile(file *models.File) error {
	query := `
		UPDATE files SET
			filename = ?, relative_path = ?, file_size = ?, modified_at = ?, hash = ?,
			mime_type = ?, width = ?, height = ?, has_thumbnail = ?, thumbnail_path = ?,
			exif_date_time = ?, exif_make = ?, exif_model = ?, exif_orientation = ?,
			exif_iso = ?, exif_fnumber = ?, exif_exposure = ?, exif_focal_length = ?,
			exif_lens = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	_, err := db.conn.Exec(query,
		file.Filename, file.RelativePath, file.FileSize, file.ModifiedAt, file.Hash,
		file.MimeType, file.Width, file.Height, file.HasThumbnail, file.ThumbnailPath,
		file.ExifDateTime, file.ExifMake, file.ExifModel, file.ExifOrientation,
		file.ExifISO, file.ExifFNumber, file.ExifExposure, file.ExifFocalLength,
		file.ExifLens, file.ID,
	)
	
	return err
}

func (db *DB) GetFileByPath(path string) (*models.File, error) {
	query := `
		SELECT id, directory_id, filename, relative_path, absolute_path, file_size, modified_at,
			hash, mime_type, width, height, has_thumbnail, thumbnail_path, created_at, updated_at,
			exif_date_time, exif_make, exif_model, exif_orientation, exif_iso, exif_fnumber,
			exif_exposure, exif_focal_length, exif_lens
		FROM files WHERE absolute_path = ?
	`
	
	var file models.File
	err := db.conn.QueryRow(query, path).Scan(
		&file.ID, &file.DirectoryID, &file.Filename, &file.RelativePath, &file.AbsolutePath,
		&file.FileSize, &file.ModifiedAt, &file.Hash, &file.MimeType, &file.Width, &file.Height,
		&file.HasThumbnail, &file.ThumbnailPath, &file.CreatedAt, &file.UpdatedAt,
		&file.ExifDateTime, &file.ExifMake, &file.ExifModel, &file.ExifOrientation,
		&file.ExifISO, &file.ExifFNumber, &file.ExifExposure, &file.ExifFocalLength, &file.ExifLens,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	
	return &file, err
}

func (db *DB) BrowseFiles(params *models.BrowseParams) (*models.BrowseResult, error) {
	baseQuery := `
		SELECT id, directory_id, filename, relative_path, absolute_path, file_size, modified_at,
			hash, mime_type, width, height, has_thumbnail, thumbnail_path, created_at, updated_at,
			exif_date_time, exif_make, exif_model, exif_orientation, exif_iso, exif_fnumber,
			exif_exposure, exif_focal_length, exif_lens
		FROM files
	`
	
	var whereClauses []string
	var args []interface{}
	
	// Build WHERE clauses
	if params.Directory != "" {
		whereClauses = append(whereClauses, "directory_id IN (SELECT id FROM directories WHERE path LIKE ?)")
		args = append(args, params.Directory+"%")
	}
	
	if params.Year > 0 {
		whereClauses = append(whereClauses, "strftime('%Y', COALESCE(exif_date_time, modified_at)) = ?")
		args = append(args, fmt.Sprintf("%04d", params.Year))
	}
	
	if params.Month > 0 {
		whereClauses = append(whereClauses, "strftime('%m', COALESCE(exif_date_time, modified_at)) = ?")
		args = append(args, fmt.Sprintf("%02d", params.Month))
	}
	
	if params.Day > 0 {
		whereClauses = append(whereClauses, "strftime('%d', COALESCE(exif_date_time, modified_at)) = ?")
		args = append(args, fmt.Sprintf("%02d", params.Day))
	}
	
	// Date range filtering
	if params.DateFrom != nil {
		whereClauses = append(whereClauses, "COALESCE(exif_date_time, modified_at) >= ?")
		args = append(args, params.DateFrom.Format("2006-01-02 15:04:05"))
	}
	
	if params.DateTo != nil {
		whereClauses = append(whereClauses, "COALESCE(exif_date_time, modified_at) <= ?")
		args = append(args, params.DateTo.Format("2006-01-02 15:04:05"))
	}
	
	if params.Search != "" {
		whereClauses = append(whereClauses, "filename LIKE ?")
		args = append(args, "%"+params.Search+"%")
	}
	
	// Build complete query
	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauses, " AND ")
	}
	
	// Order by clause
	orderBy := "ORDER BY "
	switch params.OrderBy {
	case "date":
		orderBy += "modified_at DESC"
	case "exif_date":
		orderBy += "COALESCE(exif_date_time, modified_at) DESC"
	case "name":
		orderBy += "filename ASC"
	default:
		orderBy += "relative_path ASC"
	}
	
	// Get total count
	countQuery := "SELECT COUNT(*) FROM files" + whereClause
	var total int
	err := db.conn.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, err
	}
	
	// Get files with pagination
	query := baseQuery + whereClause + " " + orderBy + " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)
	
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var files []models.File
	for rows.Next() {
		var file models.File
		err := rows.Scan(
			&file.ID, &file.DirectoryID, &file.Filename, &file.RelativePath, &file.AbsolutePath,
			&file.FileSize, &file.ModifiedAt, &file.Hash, &file.MimeType, &file.Width, &file.Height,
			&file.HasThumbnail, &file.ThumbnailPath, &file.CreatedAt, &file.UpdatedAt,
			&file.ExifDateTime, &file.ExifMake, &file.ExifModel, &file.ExifOrientation,
			&file.ExifISO, &file.ExifFNumber, &file.ExifExposure, &file.ExifFocalLength, &file.ExifLens,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	
	return &models.BrowseResult{
		Files:   files,
		Total:   total,
		HasMore: params.Offset+len(files) < total,
		Offset:  params.Offset,
	}, rows.Err()
}

func (db *DB) GetDuplicates() ([]models.FileDuplicate, error) {
	query := `
		SELECT hash, COUNT(*) as count, SUM(file_size) as total_size
		FROM files 
		WHERE hash != ''
		GROUP BY hash 
		HAVING COUNT(*) > 1
		ORDER BY count DESC, total_size DESC
	`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var duplicates []models.FileDuplicate
	for rows.Next() {
		var dup models.FileDuplicate
		err := rows.Scan(&dup.Hash, &dup.Count, &dup.TotalSize)
		if err != nil {
			return nil, err
		}
		duplicates = append(duplicates, dup)
	}
	
	return duplicates, rows.Err()
}

func (db *DB) GetFilesByHash(hash string) ([]models.File, error) {
	query := `
		SELECT id, directory_id, filename, relative_path, absolute_path, file_size, modified_at,
			hash, mime_type, width, height, has_thumbnail, thumbnail_path, created_at, updated_at,
			exif_date_time, exif_make, exif_model, exif_orientation, exif_iso, exif_fnumber,
			exif_exposure, exif_focal_length, exif_lens
		FROM files WHERE hash = ? ORDER BY relative_path
	`
	
	rows, err := db.conn.Query(query, hash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var files []models.File
	for rows.Next() {
		var file models.File
		err := rows.Scan(
			&file.ID, &file.DirectoryID, &file.Filename, &file.RelativePath, &file.AbsolutePath,
			&file.FileSize, &file.ModifiedAt, &file.Hash, &file.MimeType, &file.Width, &file.Height,
			&file.HasThumbnail, &file.ThumbnailPath, &file.CreatedAt, &file.UpdatedAt,
			&file.ExifDateTime, &file.ExifMake, &file.ExifModel, &file.ExifOrientation,
			&file.ExifISO, &file.ExifFNumber, &file.ExifExposure, &file.ExifFocalLength, &file.ExifLens,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	
	return files, rows.Err()
}

func (db *DB) GetStats() (*models.Stats, error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM directories) as total_directories,
			(SELECT COUNT(*) FROM files) as total_files,
			(SELECT SUM(file_size) FROM files) as total_size,
			(SELECT COUNT(*) FROM (SELECT hash FROM files WHERE hash != '' GROUP BY hash HAVING COUNT(*) > 1)) as duplicate_groups,
			(SELECT COUNT(*) FROM files WHERE hash IN (SELECT hash FROM files WHERE hash != '' GROUP BY hash HAVING COUNT(*) > 1)) as duplicate_files
	`
	
	var stats models.Stats
	err := db.conn.QueryRow(query).Scan(
		&stats.TotalDirectories, &stats.TotalFiles, &stats.TotalSize,
		&stats.DuplicateGroups, &stats.DuplicateFiles,
	)
	
	return &stats, err
}

func (db *DB) DeleteFile(id int) error {
	query := `DELETE FROM files WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

func (db *DB) GetFileByID(id int) (*models.File, error) {
	query := `
		SELECT id, directory_id, filename, relative_path, absolute_path, file_size, modified_at,
			hash, mime_type, width, height, has_thumbnail, thumbnail_path, created_at, updated_at,
			exif_date_time, exif_make, exif_model, exif_orientation, exif_iso, exif_fnumber,
			exif_exposure, exif_focal_length, exif_lens
		FROM files WHERE id = ?
	`
	
	var file models.File
	err := db.conn.QueryRow(query, id).Scan(
		&file.ID, &file.DirectoryID, &file.Filename, &file.RelativePath, &file.AbsolutePath,
		&file.FileSize, &file.ModifiedAt, &file.Hash, &file.MimeType, &file.Width, &file.Height,
		&file.HasThumbnail, &file.ThumbnailPath, &file.CreatedAt, &file.UpdatedAt,
		&file.ExifDateTime, &file.ExifMake, &file.ExifModel, &file.ExifOrientation,
		&file.ExifISO, &file.ExifFNumber, &file.ExifExposure, &file.ExifFocalLength, &file.ExifLens,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	
	return &file, err
}

func (db *DB) GetDirectoryTree() ([]*models.DirectoryNode, error) {
	// Get all unique subdirectories and their file counts
	subdirQuery := `
		WITH subdirs AS (
			SELECT DISTINCT
				d.path as root_path,
				d.name as root_name,
				CASE 
					WHEN f.relative_path LIKE '%/%' THEN
						substr(f.relative_path, 1, instr(f.relative_path, '/') - 1)
					ELSE NULL
				END as subdir_name,
				CASE 
					WHEN f.relative_path LIKE '%/%' THEN
						d.path || '/' || substr(f.relative_path, 1, instr(f.relative_path, '/') - 1)
					ELSE NULL
				END as subdir_path
			FROM directories d
			JOIN files f ON f.directory_id = d.id
		),
		subdir_counts AS (
			SELECT 
				subdirs.root_path,
				subdirs.subdir_path,
				subdirs.subdir_name,
				COUNT(f.id) as file_count
			FROM subdirs
			JOIN directories d ON d.path = subdirs.root_path
			JOIN files f ON f.directory_id = d.id
			WHERE subdirs.subdir_path IS NOT NULL
			AND f.relative_path LIKE subdirs.subdir_name || '/%'
			GROUP BY subdirs.root_path, subdirs.subdir_path, subdirs.subdir_name
		),
		root_counts AS (
			SELECT 
				d.path,
				d.name,
				COUNT(CASE WHEN f.relative_path NOT LIKE '%/%' THEN f.id END) as direct_files
			FROM directories d
			LEFT JOIN files f ON f.directory_id = d.id
			GROUP BY d.path, d.name
		)
		SELECT 
			root_counts.path,
			root_counts.name,
			root_counts.direct_files,
			subdir_counts.subdir_path,
			subdir_counts.subdir_name,
			COALESCE(subdir_counts.file_count, 0) as subdir_file_count
		FROM root_counts
		LEFT JOIN subdir_counts ON subdir_counts.root_path = root_counts.path
		ORDER BY root_counts.path, subdir_counts.subdir_name
	`
	
	rows, err := db.conn.Query(subdirQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// Build directory tree from results
	nodeMap := make(map[string]*models.DirectoryNode)
	var rootNodes []*models.DirectoryNode
	
	for rows.Next() {
		var rootPath, rootName, subdirPath, subdirName sql.NullString
		var directFiles, subdirFileCount int
		
		err := rows.Scan(&rootPath, &rootName, &directFiles, &subdirPath, &subdirName, &subdirFileCount)
		if err != nil {
			continue
		}
		
		// Create or get root node
		var rootNode *models.DirectoryNode
		if existing, exists := nodeMap[rootPath.String]; exists {
			rootNode = existing
		} else {
			rootNode = &models.DirectoryNode{
				Path:      rootPath.String,
				Name:      rootName.String,
				FileCount: directFiles,
				Children:  make([]*models.DirectoryNode, 0),
			}
			nodeMap[rootPath.String] = rootNode
			rootNodes = append(rootNodes, rootNode)
		}
		
		// Add subdirectory if it exists
		if subdirPath.Valid && subdirName.Valid && subdirFileCount > 0 {
			// Check if subdirectory already added
			alreadyExists := false
			for _, child := range rootNode.Children {
				if child.Path == subdirPath.String {
					alreadyExists = true
					break
				}
			}
			
			if !alreadyExists {
				subdirNode := &models.DirectoryNode{
					Path:       subdirPath.String,
					Name:       subdirName.String,
					FileCount:  subdirFileCount,
					ParentPath: rootPath.String,
					Children:   make([]*models.DirectoryNode, 0),
				}
				rootNode.Children = append(rootNode.Children, subdirNode)
				// Update root node's total file count
				rootNode.FileCount += subdirFileCount
			}
		}
	}
	
	return rootNodes, rows.Err()
}


func (db *DB) GetDateTree() ([]*models.DateTreeNode, error) {
	query := `
		SELECT 
			strftime('%Y', COALESCE(exif_date_time, modified_at)) as year,
			strftime('%m', COALESCE(exif_date_time, modified_at)) as month,
			strftime('%d', COALESCE(exif_date_time, modified_at)) as day,
			COUNT(*) as file_count
		FROM files
		WHERE year IS NOT NULL
		GROUP BY year, month, day
		ORDER BY year DESC, month DESC, day DESC
	`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// Build hierarchical structure: year -> month -> day
	yearMap := make(map[string]*models.DateTreeNode)
	var years []*models.DateTreeNode
	
	for rows.Next() {
		var year, month, day string
		var fileCount int
		
		err := rows.Scan(&year, &month, &day, &fileCount)
		if err != nil {
			continue
		}
		
		// Get or create year node
		yearNode, exists := yearMap[year]
		if !exists {
			yearNode = &models.DateTreeNode{
				Type:      "year",
				Value:     year,
				Label:     year,
				FileCount: 0,
				Children:  make([]*models.DateTreeNode, 0),
			}
			yearMap[year] = yearNode
			years = append(years, yearNode)
		}
		yearNode.FileCount += fileCount
		
		// Find or create month node
		var monthNode *models.DateTreeNode
		for _, child := range yearNode.Children {
			if child.Type == "month" && child.Value == month {
				monthNode = child
				break
			}
		}
		if monthNode == nil {
			monthName := getMonthName(month)
			monthNode = &models.DateTreeNode{
				Type:      "month",
				Value:     month,
				Label:     monthName,
				FileCount: 0,
				Children:  make([]*models.DateTreeNode, 0),
				Parent:    yearNode,
			}
			yearNode.Children = append(yearNode.Children, monthNode)
		}
		monthNode.FileCount += fileCount
		
		// Create day node
		dayNode := &models.DateTreeNode{
			Type:      "day",
			Value:     day,
			Label:     fmt.Sprintf("%s %s", monthName(month), day),
			FileCount: fileCount,
			Children:  make([]*models.DateTreeNode, 0),
			Parent:    monthNode,
		}
		monthNode.Children = append(monthNode.Children, dayNode)
	}
	
	return years, rows.Err()
}

func getMonthName(month string) string {
	months := map[string]string{
		"01": "January", "02": "February", "03": "March", "04": "April",
		"05": "May", "06": "June", "07": "July", "08": "August",
		"09": "September", "10": "October", "11": "November", "12": "December",
	}
	if name, ok := months[month]; ok {
		return name
	}
	return month
}

func monthName(month string) string {
	months := map[string]string{
		"01": "Jan", "02": "Feb", "03": "Mar", "04": "Apr",
		"05": "May", "06": "Jun", "07": "Jul", "08": "Aug",
		"09": "Sep", "10": "Oct", "11": "Nov", "12": "Dec",
	}
	if name, ok := months[month]; ok {
		return name
	}
	return month
}

func (db *DB) BrowseFilesGrouped(params *models.BrowseParams) (*models.GroupedBrowseResult, error) {
	if params.GroupBy == "" || params.GroupBy == "none" {
		// Return ungrouped results
		result, err := db.BrowseFiles(params)
		if err != nil {
			return nil, err
		}
		
		groupedResult := &models.GroupedBrowseResult{
			Groups: []models.FileGroup{{
				GroupKey:  "all",
				GroupName: "All Files",
				Files:     result.Files,
				FileCount: len(result.Files),
			}},
			Total:   result.Total,
			HasMore: result.HasMore,
			Offset:  result.Offset,
		}
		return groupedResult, nil
	}
	
	if params.GroupBy == "directory" {
		return db.browseFilesByDirectory(params)
	}
	
	return nil, fmt.Errorf("unsupported group_by parameter: %s", params.GroupBy)
}

func (db *DB) browseFilesByDirectory(params *models.BrowseParams) (*models.GroupedBrowseResult, error) {
	baseQuery := `
		SELECT 
			f.id, f.directory_id, f.filename, f.relative_path, f.absolute_path, f.file_size, f.modified_at,
			f.hash, f.mime_type, f.width, f.height, f.has_thumbnail, f.thumbnail_path, f.created_at, f.updated_at,
			f.exif_date_time, f.exif_make, f.exif_model, f.exif_orientation, f.exif_iso, f.exif_fnumber,
			f.exif_exposure, f.exif_focal_length, f.exif_lens,
			d.path as directory_path, d.name as directory_name
		FROM files f
		JOIN directories d ON f.directory_id = d.id
	`
	
	var whereClauses []string
	var args []interface{}
	
	// Apply same filters as regular browse
	if params.Directory != "" {
		whereClauses = append(whereClauses, "d.path LIKE ?")
		args = append(args, params.Directory+"%")
	}
	
	if params.Year > 0 {
		whereClauses = append(whereClauses, "strftime('%Y', COALESCE(f.exif_date_time, f.modified_at)) = ?")
		args = append(args, fmt.Sprintf("%04d", params.Year))
	}
	
	if params.Month > 0 {
		whereClauses = append(whereClauses, "strftime('%m', COALESCE(f.exif_date_time, f.modified_at)) = ?")
		args = append(args, fmt.Sprintf("%02d", params.Month))
	}
	
	if params.Day > 0 {
		whereClauses = append(whereClauses, "strftime('%d', COALESCE(f.exif_date_time, f.modified_at)) = ?")
		args = append(args, fmt.Sprintf("%02d", params.Day))
	}
	
	if params.DateFrom != nil {
		whereClauses = append(whereClauses, "COALESCE(f.exif_date_time, f.modified_at) >= ?")
		args = append(args, params.DateFrom.Format("2006-01-02 15:04:05"))
	}
	
	if params.DateTo != nil {
		whereClauses = append(whereClauses, "COALESCE(f.exif_date_time, f.modified_at) <= ?")
		args = append(args, params.DateTo.Format("2006-01-02 15:04:05"))
	}
	
	if params.Search != "" {
		whereClauses = append(whereClauses, "f.filename LIKE ?")
		args = append(args, "%"+params.Search+"%")
	}
	
	// Build complete query
	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauses, " AND ")
	}
	
	// Order by directory, then by file order preference
	orderBy := "ORDER BY d.path, "
	switch params.OrderBy {
	case "date":
		orderBy += "f.modified_at DESC"
	case "exif_date":
		orderBy += "COALESCE(f.exif_date_time, f.modified_at) DESC"
	case "name":
		orderBy += "f.filename ASC"
	default:
		orderBy += "f.relative_path ASC"
	}
	
	// Get total count
	countQuery := "SELECT COUNT(*) FROM files f JOIN directories d ON f.directory_id = d.id" + whereClause
	var total int
	err := db.conn.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, err
	}
	
	// Get files with pagination
	query := baseQuery + whereClause + " " + orderBy + " LIMIT ? OFFSET ?"
	queryArgs := append(args, params.Limit, params.Offset)
	
	rows, err := db.conn.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// Group files by directory
	groupMap := make(map[string]*models.FileGroup)
	var groupOrder []string
	
	for rows.Next() {
		var file models.File
		var directoryPath, directoryName string
		
		err := rows.Scan(
			&file.ID, &file.DirectoryID, &file.Filename, &file.RelativePath, &file.AbsolutePath,
			&file.FileSize, &file.ModifiedAt, &file.Hash, &file.MimeType, &file.Width, &file.Height,
			&file.HasThumbnail, &file.ThumbnailPath, &file.CreatedAt, &file.UpdatedAt,
			&file.ExifDateTime, &file.ExifMake, &file.ExifModel, &file.ExifOrientation,
			&file.ExifISO, &file.ExifFNumber, &file.ExifExposure, &file.ExifFocalLength, &file.ExifLens,
			&directoryPath, &directoryName,
		)
		if err != nil {
			return nil, err
		}
		
		if groupMap[directoryPath] == nil {
			groupMap[directoryPath] = &models.FileGroup{
				GroupKey:  directoryPath,
				GroupName: directoryName,
				Files:     make([]models.File, 0),
			}
			groupOrder = append(groupOrder, directoryPath)
		}
		
		groupMap[directoryPath].Files = append(groupMap[directoryPath].Files, file)
		groupMap[directoryPath].FileCount++
	}
	
	// Convert to ordered slice
	var groups []models.FileGroup
	for _, key := range groupOrder {
		groups = append(groups, *groupMap[key])
	}
	
	return &models.GroupedBrowseResult{
		Groups:  groups,
		Total:   total,
		HasMore: params.Offset+len(groups) < total,
		Offset:  params.Offset,
	}, rows.Err()
}

func (db *DB) DeleteOrphanedFiles() error {
	query := `DELETE FROM files WHERE absolute_path NOT IN (SELECT absolute_path FROM files WHERE id = files.id)`
	_, err := db.conn.Exec(query)
	return err
}
