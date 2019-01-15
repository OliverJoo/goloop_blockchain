package module

type ConsensusStatus struct {
	Height   int64
	Round    int32
	Proposer bool
}

type Consensus interface {
	Start() error
	GetStatus() *ConsensusStatus
}
