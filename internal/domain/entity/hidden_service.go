package entity

import (
	"ikedadada/go-ptor/internal/domain/value_object"
)

// HiddenService represents a hidden service in the Tor network
type HiddenService struct {
	address value_object.HiddenAddr
	relayID value_object.RelayID
	pubKey  value_object.PublicKey
}

// NewHiddenService creates a new HiddenService entity
func NewHiddenService(
	address value_object.HiddenAddr,
	relayID value_object.RelayID,
	pubKey value_object.PublicKey,
) *HiddenService {
	return &HiddenService{
		address: address,
		relayID: relayID,
		pubKey:  pubKey,
	}
}

// Address returns the hidden service address
func (hs *HiddenService) Address() value_object.HiddenAddr {
	return hs.address
}

// RelayID returns the relay ID that hosts this hidden service
func (hs *HiddenService) RelayID() value_object.RelayID {
	return hs.relayID
}

// PubKey returns the public key of the hidden service
func (hs *HiddenService) PubKey() value_object.PublicKey {
	return hs.pubKey
}

// UpdateRelay updates the relay that hosts this hidden service
func (hs *HiddenService) UpdateRelay(relayID value_object.RelayID) {
	hs.relayID = relayID
}
