package main

import (
	"log"
	"math/rand"
	"net/rpc"
	"sync/atomic"
	"time"
)

const (
	ReqAppendEntries = "AppendEntries"
	ReqRequestVote   = "RequestVote"

	VoteYes int = 1
	VoteNo  int = 0

	Accepted = 1
	Denied   = 0

	MinElectionTimeoutMS = 150 // MiliSecs

	// This will help in randomised election timeout
	MaxElectionTimeoutMSDelta = MinElectionTimeoutMS
)

const (

	// ElectionAborted means someone else became the Leader
	ElectionAborted ElectionStatus = iota
	ElectionWon                    = 1
	ElectionLoose                  = 2
)

type ElectionStatus int8

type PeerInfo struct {
	Addr string
	Id   int
}

type PeerState struct {
	cli *rpc.Client
}

type Server struct {
	Id                 int
	CurTerm            atomic.Int64
	CurState           ServerState
	cfg                Config
	heartbeatTicker    *time.Ticker
	close              chan struct{}
	abortElection      chan struct{}
	peerClients        map[int]PeerState
	alreadyVotedInTerm atomic.Bool
	log                Log
}

type Config struct {
	PeersInfo            []PeerInfo
	HeartbeatInterval    time.Duration
	AppendEntiresTimeout time.Duration
}

func (s *Server) AppendEntries(payload *RPCPayloadAppendEntries, reply *int) error {
	if len(payload.Logs) == 0 {
		// Heatbeat
		if s.IsCandidate() {
			if payload.Term >= s.CurTerm.Load() {
				s.abortElection <- struct{}{}
				*reply = Accepted
			} else {
				*reply = Denied
			}
		} else if s.IsLeader() {
			//TODO
		} else {
			s.heartbeatTicker.Reset(getRandomisedTimeout())
		}
	} else if s.acceptEntries(payload) {
		*reply = Accepted
	} else {
		*reply = Denied
	}
	return nil
}

// acceptEntries performs validity check of the request
// and commits commands  into local log
func (s *Server) acceptEntries(payload *RPCPayloadAppendEntries) bool {

	if !s.log.Found(payload.LastCommitedTerm, payload.LastCommitedLeaderIndex) {
		// Validation check failed
		return false
	}

	for _, entry := range payload.Logs {
		s.commitLog(entry, payload.Term)
	}
	return true
}

func (s *Server) RequestVote(payload *RPCPayloadRequestVote, reply *int) error {
	if s.alreadyVotedInTerm.Load() {
		*reply = VoteNo
	} else {
		*reply = VoteYes
		s.alreadyVotedInTerm.Store(true)
	}
	return nil
}

// heartbeatDetector should be run in a seperate go-routine
func (s *Server) heartbeatDetector() {
	for {
		select {
		case <-s.close:
			return
		case <-s.heartbeatTicker.C:
			// we should start elections
			s.initElection()
			if s.IsLeader() {
				s.heartbeatTicker.Stop() // stoping ticker as it's no longer required until we are leader
				return
			}
		}
	}
}

// sendHeartBeats, starts sending heart beats
func (s *Server) sendHeartBeats() {
	var doneCh = make(chan *rpc.Call, len(s.peerClients))
	s.initHeartbeatTicker()
	s.heartbeatTicker.Reset(s.cfg.HeartbeatInterval)
	for {
		select {
		case <-s.heartbeatTicker.C:
			for _, peer := range s.peerClients {
				var reply int
				peer.cli.Go(ReqAppendEntries, &RPCPayloadAppendEntries{
					Term:                    s.CurTerm.Load(),
					LastCommitedLeaderIndex: s.log.LastCommitIndex(),
				}, &reply, doneCh)
			}

		case <-s.close:
			s.heartbeatTicker.Stop()
			return
		}
	}
}

func (s *Server) initHeartbeatTicker() {
	if s.heartbeatTicker == nil {
		// It starts with election Timeout as it's duration but can be reseted any time
		s.heartbeatTicker = time.NewTicker(getRandomisedTimeout())
	}
}

// resetTerm is callback for term set
func (s *Server) resetTerm() {
	s.alreadyVotedInTerm.Store(false)
}

func (s *Server) setupPeerConns() {
	for _, peer := range s.cfg.PeersInfo {

		cli, err := rpc.Dial("tcp", peer.Addr)
		if err != nil {
			log.Print("error while establishing connection with peer", peer.Addr)
		}
		s.peerClients[peer.Id] = PeerState{
			cli: cli,
		}
	}
}

func (s *Server) NextState(action Action) {

}

func (s *Server) IsFollower() bool {
	return s.CurState == Follower
}

func (s *Server) IsCandidate() bool {
	return s.CurState == Candidate
}

func (s *Server) IsLeader() bool {
	return s.CurState == Leader
}

func getRandomisedTimeout() time.Duration {

	// We are adding this jitter to avoid perodic split vote condition
	electionTimeout := MinElectionTimeoutMS + rand.Intn(MaxElectionTimeoutMSDelta)
	timeout := time.Duration(electionTimeout) * time.Millisecond
	return timeout
}

func (s *Server) WonQurom(votes int) bool {
	clusterSize := len(s.peerClients) + 1 // Adding one for leader
	return votes >= clusterSize/2
}
