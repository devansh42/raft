package main

import (
	"net/rpc"
	"time"
)

func (s *Server) startReplication(cmd []byte, confirmations chan struct{}) {
	for _, peer := range s.peerClients {
		s.replicate(cmd, peer.cli, nil, confirmations)
	}
}

func (s *Server) replicate(cmd []byte, cli *rpc.Client, timer *time.Timer, confirmations chan struct{}) {
	var reply int
	// We want to minimise timers that's why we are creating when necessary only
	if timer == nil {
		timer = time.NewTimer(s.cfg.AppendEntiresTimeout)
	} else {
		timer.Reset(s.cfg.AppendEntiresTimeout)
	}
	call := cli.Go(ReqAppendEntries, &RPCPayloadAppendEntries{
		Term: s.CurTerm.Load(),
		// TODO:  For a slow server his might change during multiple retries
		// Is it safe?
		LastCommitedLeaderIndex: s.log.LastCommitIndex(),
		LastCommitedTerm:        s.log.LastCommitedTerm(),
		Logs:                    [][]byte{cmd},
	}, &reply, nil)
	go s.trackReplication(call, cmd, cli, timer, confirmations)
}

func (s *Server) trackReplication(call *rpc.Call, cmd []byte, cli *rpc.Client, timer *time.Timer, confirmations chan struct{}) {
	select {
	case <-call.Done:
		// Done
		timer.Stop()
		reply := call.Reply.(*int)
		if *reply == Accepted {
			confirmations <- struct{}{}
		}
		// TODO: Process No
		return
	case <-s.close:
		// Just die
		return
	case <-timer.C:
		// Timeout, Retry
		// TODO: Should we retry with exponential backoff
		s.replicate(cmd, cli, timer, confirmations)
	}
}
