package repository

import (
	"ikedadada/go-ptor/internal/domain/entity"
	vo "ikedadada/go-ptor/internal/domain/value_object"
)

// HiddenServiceRepository manages hidden service entities
type HiddenServiceRepository interface {
	// FindByAddress finds a hidden service by its address
	FindByAddress(address vo.HiddenAddr) (*entity.HiddenService, error)
	
	// FindByAddressString finds a hidden service by its string address (case-insensitive)
	FindByAddressString(address string) (*entity.HiddenService, error)
	
	// All returns all hidden services
	All() ([]*entity.HiddenService, error)
	
	// Save stores a hidden service
	Save(hiddenService *entity.HiddenService) error
}