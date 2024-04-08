package metadata

import (
	"encoding/json"
	"os"
	"time"
)

type CreatorInfo struct {
	Name     string `json:"name"`
	Service  string `json:"service"`
	PageLink string `json:"pageLink"`
}

type Metadata struct {
	Creator CreatorInfo `json:"creator"`
	Files   []FileInfo  `json:"files"`
}

type FileInfo struct {
	FileName    string    `json:"fileName"`
	Size        int64     `json:"size"`
	Description string    `json:"description"`
	DownloadURL string    `json:"downloadURL"`
	Published   time.Time `json:"published"`
}

func AppendMetadata(filePath string, fileInfo FileInfo, creator CreatorInfo) error {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	var metadata Metadata
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if stat.Size() != 0 {
		if err := json.NewDecoder(file).Decode(&metadata); err != nil {
			return err
		}
	}

	metadata.Creator = creator

	metadata.Files = append(metadata.Files, fileInfo)

	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	if err := json.NewEncoder(file).Encode(metadata); err != nil {
		return err
	}

	return nil
}

func URLExistsInMetadata(metadataFilePath, url string) (bool, error) {
	file, err := os.Open(metadataFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	var metadata Metadata
	if err := json.NewDecoder(file).Decode(&metadata); err != nil {
		return false, err
	}

	for _, fileInfo := range metadata.Files {
		if fileInfo.DownloadURL == url {
			return true, nil
		}
	}

	return false, nil
}

func ReadMetadata(filePath string) (*Metadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var metadata Metadata
	if err := json.NewDecoder(file).Decode(&metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}
