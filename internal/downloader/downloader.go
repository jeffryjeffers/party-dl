package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"party-dl/internal/metadata"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Downloader struct {
	BaseDir string
	Creator metadata.CreatorInfo
}

func NewDownloader(baseDir string, creator metadata.CreatorInfo) *Downloader {
	return &Downloader{
		BaseDir: baseDir,
		Creator: creator,
	}
}

func (d *Downloader) DownloadURL(url, description string, published time.Time) (string, bool, error) {
	createDirectories(d.BaseDir)

	dir := getDirectoryForExtension(filepath.Ext(url))

	dirPath := filepath.Join(d.BaseDir, dir)
	createDirectory(dirPath)

	fileName := generateUniqueFileName(filepath.Ext(url))

	if exists, err := metadata.URLExistsInMetadata(filepath.Join(d.BaseDir, "metadata.json"), url); err != nil {
		return "", false, err
	} else if exists {
		return "", true, nil
	}

	partFilePath := filepath.Join(dirPath, fileName+".part")
	if err := downloadFile(url, partFilePath); err != nil {
		os.Remove(partFilePath)
		return "", false, err
	}

	filePath := filepath.Join(dirPath, fileName)
	if err := os.Rename(partFilePath, filePath); err != nil {
		// If renaming fails, delete the part file
		os.Remove(partFilePath)
		return "", false, err
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", false, err
	}
	size := fileInfo.Size()

	fileInfoStruct := metadata.FileInfo{
		FileName:    fileName,
		Size:        size,
		Description: description,
		Published:   published,
		DownloadURL: url,
	}

	metadataFilePath := filepath.Join(d.BaseDir, "metadata.json")
	if err := metadata.AppendMetadata(metadataFilePath, fileInfoStruct, d.Creator); err != nil {
		return "", false, err
	}

	return filePath, false, nil
}

func downloadFile(url, filePath string) error {
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s", response.Status)
	}

	_, err = io.Copy(out, response.Body)
	if err != nil {
		os.Remove(filePath)
		return err
	}

	return nil
}

func createDirectories(baseDir string) {
	directories := []string{"images", "videos"}
	for _, dir := range directories {
		dirPath := filepath.Join(baseDir, dir)
		createDirectory(dirPath)
	}
}

func createDirectory(dirPath string) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		os.MkdirAll(dirPath, os.ModePerm)
	}
}

func getDirectoryForExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png", ".gif":
		return "images"
	case ".mp4", ".mov", ".avi", ".mkv":
		return "videos"
	default:
		return "other"
	}
}

func generateUniqueFileName(extension string) string {
	u := uuid.New()
	return strings.ReplaceAll(u.String(), "-", "") + extension
}
