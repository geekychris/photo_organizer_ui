# Photo Database Manager

A comprehensive photo management application with a web UI for cataloging, browsing, and managing photo collections. Built in Go with SQLite database backend.

## Features

- **Directory Scanning**: Recursively scan directories for photo files
- **EXIF Data Extraction**: Extract and store camera metadata (make, model, ISO, f-stop, etc.)
- **Duplicate Detection**: Find duplicate images using SHA256 file hashing
- **Thumbnail Generation**: Automatic thumbnail creation for fast browsing
- **Web Interface**: Clean, responsive web UI for browsing and managing photos
- **Multiple View Modes**: Browse by directory structure or chronologically
- **Infinite Scroll**: Smooth browsing experience without pagination
- **Search & Filtering**: Search by filename and filter by date

## Supported Formats

- JPEG (.jpg, .jpeg)
- PNG (.png)
- GIF (.gif)
- WebP (.webp)
- TIFF (.tiff, .tif)
- BMP (.bmp)
- RAW formats (.raw, .cr2, .nef, .arw, .dng)

## Installation

1. **Clone and build**:
   ```bash
   go build
   ```

2. **Run the application**:
   ```bash
   ./photodatabase -serve
   ```

## Usage

### Command Line Options

```bash
# Start the web server (default port 8080)
./photodatabase -serve

# Scan a directory
./photodatabase -scan /path/to/photos

# Rescan all managed directories
./photodatabase -rescan

# Use custom configuration file
./photodatabase -config myconfig.yaml

# Use custom port
./photodatabase -serve -port 9000
```

### Configuration

Create a `config.yaml` file to customize settings:

```yaml
database_path: "photos.db"      # SQLite database file
thumbnail_dir: "thumbnails"     # Thumbnail storage directory
thumbnail_size: 300             # Thumbnail size in pixels
web_port: "8080"               # Web server port
```

### Web Interface

Once the server is running, open your browser to `http://localhost:8080`:

1. **Dashboard**: Overview of your photo collection with statistics
2. **Browse Photos**: View photos in thumbnail or detail view
3. **Duplicates**: Find and manage duplicate files
4. **Directories**: Manage scanned directories

### Basic Workflow

1. Start the web server: `./photodatabase -serve`
2. Open http://localhost:8080 in your browser
3. Click "Scan Directory" to add photo directories
4. Browse your photos using the web interface
5. Use the duplicates page to find and remove duplicate files

## Database Schema

The application uses SQLite with the following main tables:

- `directories`: Manages root-level directories being monitored
- `files`: Stores file metadata, EXIF data, and thumbnail information

## Key Features Explained

### Directory Scanning
- Recursively walks through directories
- Extracts EXIF data from image files
- Calculates SHA256 hashes for duplicate detection
- Tracks file modification times for efficient rescanning

### EXIF Data
- Camera make and model
- Date/time taken (preferred over file modification time)
- Camera settings (ISO, f-stop, exposure time, focal length)
- Lens information (when available)

### Directory Tree Navigation
- **Hierarchical directory browsing**: View photos organized by their folder structure
- **Directory tree sidebar**: Navigate through nested directories with file counts
- **Breadcrumb navigation**: Track your current location in the directory structure
- **Directory grouping**: View photos grouped by their containing directories
- **Two view modes**: Switch between flat photo view and directory-organized view

### Advanced Filtering
- **Date range filtering**: Select photos between specific dates using date pickers
- **Individual date filtering**: Filter by specific year, month, or day
- **Search by filename**: Find photos by their filename
- **Directory filtering**: Browse photos from specific directories
- **Multiple sort options**: Sort by directory path, file date, EXIF date, or filename

### Duplicate Detection
- Uses SHA256 file hashing for accurate duplicate identification
- Groups duplicates by hash with size and count information
- Shows potential space savings from removing duplicates
- Provides detailed file location information

### Thumbnail Generation
- On-demand thumbnail creation
- Cached thumbnail storage
- Maintains aspect ratios
- JPEG compression for storage efficiency

### Web Interface
- Bootstrap-based responsive design
- Infinite scroll for smooth browsing
- **Directory tree sidebar** with collapsible navigation
- **Advanced filters** with date range pickers
- **View mode toggles** between thumbnail and detail views
- Modal viewer for detailed photo information

## API Endpoints

The application provides a REST API:

### Core Endpoints
- `GET /api/stats` - Collection statistics
- `GET /api/files` - Browse files with filters
- `GET /api/files/grouped` - Browse files grouped by directory
- `GET /api/files/{id}` - Get specific file details
- `GET /api/duplicates` - Get duplicate file groups
- `GET /api/duplicates/{hash}` - Get files with specific hash

### Directory Management
- `GET /api/directories` - List managed directories
- `GET /api/directories/tree` - Get directory tree structure
- `POST /api/scan` - Start directory scan
- `POST /api/rescan` - Rescan all directories

### File Serving
- `GET /image/{id}` - Serve original image
- `GET /thumbnail/{id}` - Serve thumbnail

### Query Parameters for /api/files and /api/files/grouped
- `limit` - Number of results (max 200, default 50)
- `offset` - Pagination offset
- `order_by` - Sort order: "path", "date", "exif_date", "name"
- `group_by` - Grouping: "directory", "none" (grouped endpoint only)
- `date_from` - Start date (YYYY-MM-DD format)
- `date_to` - End date (YYYY-MM-DD format)
- `year` - Specific year filter
- `month` - Specific month filter (1-12)
- `day` - Specific day filter (1-31)
- `directory` - Filter by directory path
- `search` - Search in filenames

## Performance Considerations

- **Large Collections**: The application is designed to handle large photo collections efficiently
- **Incremental Scanning**: Only rescans files that have been modified
- **Database Indexing**: Proper indexes for fast queries
- **Thumbnail Caching**: Thumbnails are generated once and cached

## Development

To modify or extend the application:

```bash
# Install dependencies
go mod tidy

# Build and run
go build && ./photodatabase -serve
```

Project structure:
- `internal/config/` - Configuration management
- `internal/database/` - Database operations
- `internal/models/` - Data models
- `internal/scanner/` - Directory scanning and EXIF extraction
- `internal/server/` - Web server and API
- `web/templates/` - HTML templates
- `web/static/` - CSS and JavaScript

## License

This project is open source. Feel free to modify and distribute according to your needs.

## Troubleshooting

### Common Issues

**Database locked errors**:
- Ensure only one instance is running
- Check file permissions on the database file

**Missing thumbnails**:
- Verify the thumbnail directory is writable
- Check that image files are accessible

**EXIF data not showing**:
- Some image formats may not contain EXIF data
- RAW files may need additional processing

**Performance issues**:
- Consider running rescans during off-peak hours
- Ensure adequate disk space for thumbnails
- Monitor memory usage with very large collections