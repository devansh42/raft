package main

import (
	"errors"
	"sort"
)

var ErrNotLeader = errors.New("I'm not the leader")

type Entry struct {
	Term  int64
	Cmd   []byte
	Index int64
}

type Log struct {
	entires           []Entry
	lastCommitedIndex int64
	lastCommitedTerm  int64
}

func (l *Log) Add(entry Entry) {
	// TODO: Add Locking
	l.entires = append(l.entires, entry)
}

func (l *Log) SetLastCommitIndex(index int64) {
	l.lastCommitedIndex = index
}

func (l *Log) SetLastCommitedTerm(term int64) {
	l.lastCommitedTerm = term
}

func (l *Log) LastCommitIndex() int64 {
	return l.lastCommitedIndex
}
func (l *Log) LastCommitedTerm() int64 {
	return l.lastCommitedTerm
}
func (l *Log) IncrementCommitIndex() int64 {
	l.lastCommitedIndex += 1
	return l.lastCommitedIndex
}

func (l *Log) Found(term, index int64) bool {
	pos := sort.Search(len(l.entires), func(i int) bool {

		return l.entires[i].Index >= index && l.entires[i].Term >= term
	})
	if pos == len(l.entires) {
		return false
	}
	return l.entires[pos].Index == index && l.entires[pos].Term == term
}

func (s *Server) ApplyCommand(cmd []byte) error {
	if !s.IsLeader() {
		return ErrNotLeader
	}

	// This channel accumalted all the ayes
	//TODO:  We never close this channel, Is it a good idea?
	var confirmations = make(chan struct{})
	term := s.CurTerm.Load()
	s.startReplication(cmd, confirmations)
	var totalConfirms = 1 // 1 for Leader himself
	for range confirmations {
		totalConfirms += 1
		if s.WonQurom(totalConfirms) {
			s.commitLog(cmd, term)
			// We are leaving at completion of qurom
			// it's possible that some slow or faulty servers are still in the replication process
			return nil
		}
	}

	// TODO: Should we return some error
	return nil
}

// commitLog adds cmd to local log for the server
// TODO: Add a persistence storage layer
func (s *Server) commitLog(cmd []byte, term int64) {
	s.log.Add(Entry{
		Term:  term,
		Cmd:   cmd,
		Index: s.log.IncrementCommitIndex(),
	})
}
