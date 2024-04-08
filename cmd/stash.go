package cmd

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"os"
	"party-dl/internal/metadata"
	stash2 "party-dl/internal/stash"
	"path/filepath"
	"time"
)

var (
	stashHost = ""
	content   = ""
)

func stashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stash --stash-host http://localhost:9999 --content ./data/",
		Short:   "add metadata to stash",
		Example: "party-dl stash --stash-host http://localhost:9999 --content ./data/",
		Aliases: []string{"s", "stash"},
		RunE:    stash,
	}
	cmd.Flags().StringVarP(&stashHost, "stash-host", "", "", "Stash host")
	cmd.Flags().StringVarP(&content, "content", "c", "", "Path to your stash content folder")
	return cmd
}

func stash(cmd *cobra.Command, args []string) error {
	if stashHost == "" {
		log.Error("no stash host specified")
	}
	if content == "" {
		log.Error("no content specified")
	}

	stashManager := stash2.NewManager(stashHost)

	log.Infof("Searching for metadata files...")

	metaFiles, err := findMetadataJSONFiles(content)
	if err != nil {
		log.Error(err)
		return nil
	}

	log.Infof("Found %d metadata files", len(metaFiles))

	for _, metaFile := range metaFiles {
		log.Infof("Adding metadata to stash: %s", metaFile)
		meta, err := metadata.ReadMetadata(metaFile)
		if err != nil {
			log.Error(err)
			continue
		}
		studioName := ""
		studioUrl := ""
		switch meta.Creator.Service {
		case "onlyfans":
			studioName = "OnlyFans"
			studioUrl = "https://onlyfans.com"
		case "fansly":
			studioName = "Fansly"
			studioUrl = "https://fansly.com"
		}
		studioId, err := stashManager.GetOrCreateStudio(studioName, studioUrl)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Info("Found/Created studio", "id", studioId, "name", studioName, "url", studioUrl)

		performerId, err := stashManager.GetOrCreatePerformer(meta.Creator.Name, meta.Creator.PageLink)
		if err != nil {
			log.Error(err)
			continue
		}
		log.Info("Found/Created performer", "id", performerId, "name", meta.Creator.Name, "url", meta.Creator.PageLink)

		for _, file := range meta.Files {
			scene, found, err := stashManager.GetSceneByPathAndSize(file.FileName, file.Size)
			if err != nil {
				log.Error(err)
				continue
			}
			if !found {
				continue
			}
			updateInput := stash2.UpdateInput{
				ID:          scene[5].(float64),
				Date:        file.Published.Format(time.RFC3339),
				StudioID:    studioId,
				Details:     file.Description,
				Title:       fmt.Sprintf("%s - %s", meta.Creator.Name, file.Published.Format(time.DateOnly)),
				PerformerID: performerId,
			}
			_, err = stashManager.UpdateScene(updateInput)
			if err != nil {
				log.Error(err)
				continue
			}
			log.Infof("Added metadata to scene: %s", file.FileName)
		}

		for _, file := range meta.Files {
			scene, found, err := stashManager.GetImageByPathAndSize(file.FileName, file.Size)
			if err != nil {
				log.Error(err)
				continue
			}
			if !found {
				continue
			}
			updateInput := stash2.UpdateInput{
				ID:          scene[5].(float64),
				Date:        file.Published.Format(time.RFC3339),
				StudioID:    studioId,
				Details:     file.Description,
				Title:       fmt.Sprintf("%s - %s", meta.Creator.Name, file.Published.Format(time.DateOnly)),
				PerformerID: performerId,
			}
			_, err = stashManager.UpdateImage(updateInput)
			if err != nil {
				log.Error(err)
				continue
			}
			log.Infof("Added metadata to image: %s", file.FileName)
		}

	}

	return nil
}

func findMetadataJSONFiles(directory string) ([]string, error) {
	var metadataFiles []string

	subdirs, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	for _, subdir := range subdirs {
		if subdir.IsDir() {
			subdirPath := filepath.Join(directory, subdir.Name())

			files, err := os.ReadDir(subdirPath)
			if err != nil {
				return nil, err
			}

			for _, file := range files {
				if !file.IsDir() && file.Name() == "metadata.json" {
					metadataFiles = append(metadataFiles, filepath.Join(subdirPath, file.Name()))
				}
			}
		}
	}

	return metadataFiles, nil
}
