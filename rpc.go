package main

type RPCPayloadRequestVote struct {
	Id int
}
type RPCPayloadAppendEntries struct {
	Logs                    [][]byte
	Term                    int64
	LastCommitedLeaderIndex int64
	LastCommitedTerm        int64
}
