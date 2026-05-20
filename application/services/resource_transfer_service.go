package services

import (
	"github.com/drama-generator/backend/pkg/logger"
	"gorm.io/gorm"
)

type ResourceTransferService struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewResourceTransferService(db *gorm.DB, log *logger.Logger) *ResourceTransferService {
	return &ResourceTransferService{
		db:  db,
		log: log,
	}
}

// ResourceTransferService now only retains basic structure, MinIO-related features have been removed
// For resource transfer functionality, please use local storage
