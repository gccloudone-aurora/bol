package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"time"

	"github.com/gccloudone-aurora/bol/pkg/storage/azurestorage"
	"github.com/gccloudone-aurora/bol/pkg/util"
)

// Storage is an interface that defines methods for storing and retrieving artifacts.
type Storage interface {
	UploadArtifact(ctx context.Context, fileName string, data io.Reader) error
	LastDateOfFileUploaded(ctx context.Context, fileNameRegex *regexp.Regexp) (*time.Time, error)
}

func NewStorage(config util.ArtifactRepository) (Storage, error) {
	switch config.Provider {
	case "azure":
		azureStorage, err := azurestorage.NewAzureStorage(config.Azure.StorageAccountName, config.Azure.StorageAccountContainerName, config.Azure.UseAzureCliCredentials)
		if err != nil {
			log.Fatalf("Failed to create Azure storage client: %v", err)
		}
		return azureStorage, nil
	}

	return nil, fmt.Errorf("Unsupported storage provider: %w", config.Provider)
}
