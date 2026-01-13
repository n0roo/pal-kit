package marker

// Example file to test PAL marker parsing
// This file contains sample markers for testing purposes.

// Session Service handles session operations
// @pal-port L1-SessionService
// @pal-layer l1
// @pal-domain sessions
// @pal-generated
type SessionService struct{}

// Port Composite Service
// @pal-port L2-PortCompositeService
// @pal-layer l2
// @pal-domain ports
// @pal-depends L1-SessionService, L1-PortQueryService
// @pal-generated
type PortCompositeService struct{}

// L1-PortQueryService is a query service
// @pal-port L1-PortQueryService
// @pal-layer l1
// @pal-domain ports
type PortQueryService struct{}
