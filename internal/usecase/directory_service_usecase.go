package usecase

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"ikedadada/go-ptor/internal/domain/entity"
)

// DirectoryServiceUseCase handles fetching directory information from directory servers
type DirectoryServiceUseCase interface {
	FetchDirectory(input DirectoryServiceInput) (DirectoryServiceOutput, error)
	FetchRelays(input DirectoryServiceInput) (RelayServiceOutput, error)
	FetchHiddenServices(input DirectoryServiceInput) (HiddenServiceOutput, error)
}

type DirectoryServiceInput struct {
	BaseURL string
}

type DirectoryServiceOutput struct {
	Directory entity.Directory
}

type RelayServiceOutput struct {
	Relays map[string]entity.RelayInfo
}

type HiddenServiceOutput struct {
	HiddenServices map[string]entity.HiddenServiceInfo
}

type directoryServiceUseCaseImpl struct {
	httpClient *http.Client
}

func NewDirectoryServiceUseCase() DirectoryServiceUseCase {
	return &directoryServiceUseCaseImpl{
		httpClient: &http.Client{},
	}
}

func (uc *directoryServiceUseCaseImpl) FetchDirectory(input DirectoryServiceInput) (DirectoryServiceOutput, error) {
	relayOut, err := uc.FetchRelays(input)
	if err != nil {
		return DirectoryServiceOutput{}, fmt.Errorf("fetch relays failed: %w", err)
	}

	hiddenOut, err := uc.FetchHiddenServices(input)
	if err != nil {
		return DirectoryServiceOutput{}, fmt.Errorf("fetch hidden services failed: %w", err)
	}

	directory := entity.Directory{
		Relays:         relayOut.Relays,
		HiddenServices: hiddenOut.HiddenServices,
	}

	return DirectoryServiceOutput{Directory: directory}, nil
}

func (uc *directoryServiceUseCaseImpl) FetchRelays(input DirectoryServiceInput) (RelayServiceOutput, error) {
	url := strings.TrimRight(input.BaseURL, "/") + "/relays.json"
	log.Printf("request GET %s", url)

	res, err := uc.httpClient.Get(url)
	if err != nil {
		return RelayServiceOutput{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	log.Printf("response GET %s status=%s", url, res.Status)

	if res.StatusCode != http.StatusOK {
		return RelayServiceOutput{}, fmt.Errorf("unexpected status: %s", res.Status)
	}

	var d entity.Directory
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		return RelayServiceOutput{}, fmt.Errorf("decode JSON failed: %w", err)
	}

	return RelayServiceOutput{Relays: d.Relays}, nil
}

func (uc *directoryServiceUseCaseImpl) FetchHiddenServices(input DirectoryServiceInput) (HiddenServiceOutput, error) {
	url := strings.TrimRight(input.BaseURL, "/") + "/hidden.json"
	log.Printf("request GET %s", url)

	res, err := uc.httpClient.Get(url)
	if err != nil {
		return HiddenServiceOutput{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer res.Body.Close()

	log.Printf("response GET %s status=%s", url, res.Status)

	if res.StatusCode != http.StatusOK {
		return HiddenServiceOutput{}, fmt.Errorf("unexpected status: %s", res.Status)
	}

	var d entity.Directory
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		return HiddenServiceOutput{}, fmt.Errorf("decode JSON failed: %w", err)
	}

	// Normalize hidden service keys to lowercase for case-insensitive lookup
	normalized := make(map[string]entity.HiddenServiceInfo, len(d.HiddenServices))
	for k, v := range d.HiddenServices {
		normalized[strings.ToLower(k)] = v
	}

	return HiddenServiceOutput{HiddenServices: normalized}, nil
}