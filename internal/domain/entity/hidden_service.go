package entity

import (
	vo "ikedadada/go-ptor/internal/domain/value_object"
)

// HiddenService represents a hidden service in the Tor network
type HiddenService struct {
	address vo.HiddenAddr
	relayID vo.RelayID
	pubKey  vo.PublicKey
}

// NewHiddenService creates a new HiddenService entity
func NewHiddenService(
	address vo.HiddenAddr,
	relayID vo.RelayID,
	pubKey vo.PublicKey,
) *HiddenService {
	return &HiddenService{
		address: address,
		relayID: relayID,
		pubKey:  pubKey,
	}
}

// Address returns the hidden service address
func (hs *HiddenService) Address() vo.HiddenAddr {
	return hs.address
}

// RelayID returns the relay ID that hosts this hidden service
func (hs *HiddenService) RelayID() vo.RelayID {
	return hs.relayID
}

// PubKey returns the public key of the hidden service
func (hs *HiddenService) PubKey() vo.PublicKey {
	return hs.pubKey
}

// UpdateRelay updates the relay that hosts this hidden service
func (hs *HiddenService) UpdateRelay(relayID vo.RelayID) {
	hs.relayID = relayID
}
