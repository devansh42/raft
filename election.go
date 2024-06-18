package main

import (
	"net/rpc"
	"time"
)

func (s *Server) initElection() {
	s.NextState(MissedHeartbeat)
	var totalVotes = 1 // Server votes for himself
	var electionTimer *time.Timer
	for {

		s.CurTerm.Add(1)
		s.resetTerm()

		timeout := getRandomisedTimeout()
		if electionTimer == nil {
			electionTimer = time.NewTimer(timeout)
		} else {
			electionTimer.Reset(timeout)
		}

		result := s.beingElection(electionTimer, totalVotes)
		switch result {
		case ElectionLoose:
			s.NextState(ElectionGotLoose)
			continue // Continue boy, give it another shot
		case ElectionAborted:
			electionTimer.Stop()
			s.NextState(ElectionGotAbort)
			return // Someone else become leader
		case ElectionWon:
			electionTimer.Stop()
			s.NextState(ElectionGotWin)
			return // We won
		}
	}
}

// beingElection asks every peer for vote and then waits for there reply and process reply
// or returns when election timeout or election aborts because of some new leader
func (s *Server) beingElection(electionTime *time.Timer, totalVotes int) ElectionStatus {
	var voteCh = make(chan *rpc.Call, len(s.peerClients))
	for _, v := range s.peerClients {
		var reply int
		v.cli.Go(ReqRequestVote, &RPCPayloadRequestVote{Id: s.Id}, &reply, voteCh)
	}

	for range s.peerClients {
		select {
		case resp := <-voteCh:
			recivedVote := resp.Reply.(*int)
			if *recivedVote == VoteYes {
				totalVotes++
				if totalVotes >= (len(s.peerClients)+1)/2 {

					// We won the election
					// We are the boss now
					return ElectionWon
				}
			}
		case <-electionTime.C:
			// Election Timeout
			// Here we go again
			return ElectionLoose

		case <-s.abortElection:
			// Election aborted
			return ElectionAborted
		}
	}
	return ElectionLoose
}
