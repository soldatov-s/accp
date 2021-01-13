package metrics

type IMetrics interface {
	// GetMetrics return map of the metrics from provider
	GetAllMetrics(out MapMetricsOptions) (MapMetricsOptions, error)
	// GetAliveHandlers return array of the aliveHandlers from provider
	GetAllAliveHandlers(out MapCheckFunc) (MapCheckFunc, error)
	// GetReadyHandlers return array of the readyHandlers from provider
	GetAllReadyHandlers(out MapCheckFunc) (MapCheckFunc, error)
}

type Service struct {
	// Connection metrics
	Metrics MapMetricsOptions
	// Connection alive handlers
	AliveHandlers MapCheckFunc
	// Connection ready handlers
	ReadyHandlers MapCheckFunc
}

// GetAliveHandlers return array of the aliveHandlers from service
func (s *Service) GetAliveHandlers() MapCheckFunc {
	if s.AliveHandlers == nil {
		s.AliveHandlers = make(MapCheckFunc)
	}

	return s.AliveHandlers
}

// GetMetrics return map of the metrics from cache connection
func (s *Service) GetMetrics() MapMetricsOptions {
	if s.Metrics == nil {
		s.Metrics = make(MapMetricsOptions)
	}

	return s.Metrics
}

// GetReadyHandlers return array of the readyHandlers from cache connection
func (s *Service) GetReadyHandlers() MapCheckFunc {
	if s.ReadyHandlers == nil {
		s.ReadyHandlers = make(MapCheckFunc)
	}

	return s.ReadyHandlers
}
