package usecase

import (
	"fmt"
	"net"
	"strings"

	"ikedadada/go-ptor/shared/domain/repository"
)

// ResolveTargetAddressInput specifies the target to resolve
type ResolveTargetAddressInput struct {
	Host string
	Port int
}

// ResolveTargetAddressOutput contains resolved address and exit relay information
type ResolveTargetAddressOutput struct {
	DialAddress string
	ExitRelayID string
}

// ResolveTargetAddressUseCase resolves target addresses, handling hidden services
type ResolveTargetAddressUseCase interface {
	Handle(in ResolveTargetAddressInput) (ResolveTargetAddressOutput, error)
}

type resolveTargetAddressUseCaseImpl struct {
	hsRepo repository.HiddenServiceRepository
}

// NewResolveTargetAddressUseCase creates a new use case for address resolution
func NewResolveTargetAddressUseCase(hsRepo repository.HiddenServiceRepository) ResolveTargetAddressUseCase {
	return &resolveTargetAddressUseCaseImpl{hsRepo: hsRepo}
}

func (uc *resolveTargetAddressUseCaseImpl) Handle(in ResolveTargetAddressInput) (ResolveTargetAddressOutput, error) {
	hostLower := strings.ToLower(in.Host)
	exitRelayID := ""

	// Handle .ptor hidden service addresses
	if strings.HasSuffix(hostLower, ".ptor") {
		hs, err := uc.hsRepo.FindByAddressString(hostLower)
		if err != nil {
			return ResolveTargetAddressOutput{}, fmt.Errorf("hidden service not found: %s", in.Host)
		}
		exitRelayID = hs.RelayID().String()
	}

	// Format dial address based on IP type
	var dialAddress string
	if ip := net.ParseIP(hostLower); ip != nil && ip.To4() == nil {
		// IPv6 address needs brackets
		dialAddress = fmt.Sprintf("[%s]:%d", hostLower, in.Port)
	} else {
		// IPv4 address or hostname
		dialAddress = fmt.Sprintf("%s:%d", hostLower, in.Port)
	}

	return ResolveTargetAddressOutput{
		DialAddress: dialAddress,
		ExitRelayID: exitRelayID,
	}, nil
}
