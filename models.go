package main

type ServerState int
type Action int

const (
	Follower ServerState = iota
	Candidate
	Leader
)

const (
	Init Action = iota
	ElectionStarted
	ElectionGotAbort
	ElectionGotWin
	ElectionGotTimeout
	ElectionGotLoose
	MissedHeartbeat
)

func (s *ServerState) NextState(action Action) ServerState {
	// TODO: Implement

	return 0
}
