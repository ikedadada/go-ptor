package entity

// Directory represents a response from the directory service.
type Directory struct {
	Relays         map[string]RelayInfo         `json:"relays"`
	HiddenServices map[string]HiddenServiceInfo `json:"hidden_services"`
}

// RelayInfo contains metadata for a relay node published by the directory.
type RelayInfo struct {
	Endpoint string `json:"endpoint"`
	PubKey   string `json:"pubkey"`
}

// HiddenServiceInfo maps a hidden service address to its relay and public key.
type HiddenServiceInfo struct {
	Relay  string `json:"relay"`
	PubKey string `json:"pubkey"`
}
