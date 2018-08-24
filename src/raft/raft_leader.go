package raft

import (
	"sort"
	"time"
)

func (rf *Raft) initLeader() {
	rf.mu.Lock()
	// re-initialise nextIndex, matchIndex
	rf.nextIndex = make([]int, len(rf.peers))
	for i := range rf.nextIndex {
		rf.nextIndex[i] = len(rf.log)
	}
	rf.matchIndex = make([]int, len(rf.peers))
	rf.mu.Unlock()
	beatNumber := 1
	rf.heartbeat(beatNumber)
}

func (rf *Raft) heartbeat(beatNumber int) {
	P("H>", rf.me, rf.currentTerm, beatNumber)
	startTime := time.Now() // for benchmarking

	rf.mu.Lock()
	// leader check
	if rf.phase != "leader" {
		P(rf.me, "x leader (1)")
		rf.mu.Unlock()
		return
	}
	// send heartbeats
	replies := make([]AppendEntriesReply, len(rf.peers))
	myLastIndex := len(rf.log) - 1 // why is this here?????
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID != rf.me {
			go func(ID int) {
				nextIndex := rf.nextIndex[ID]
				args := AppendEntriesArgs{
					Term:         rf.currentTerm,
					LeaderID:     rf.me,
					LeaderCommit: rf.commitIndex,
					PrevLogIndex: Min(nextIndex, len(rf.log)) - 1, // nextIndex ≥ 1
				}
				args.PrevLogTerm = rf.log[args.PrevLogIndex].Term
				if nextIndex < len(rf.log) {
					args.Entries = rf.log[nextIndex:]
				}
				rf.sendAppendEntries(ID, &args, &replies[ID])
			}(ID)
		}
	}
	rf.mu.Unlock()

	// heartbeat timeout
	time.Sleep(heartbeatTimeout)

	// leader check 2
	rf.mu.Lock()
	if rf.phase != "leader" {
		P(rf.me, "x leader (2)")
		rf.mu.Unlock()
		return
	}

	// adjusting follower indices depending on responses
	P("H<", rf.me, rf.currentTerm, beatNumber)
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID != rf.me {
			if replies[ID].Success { // implicity times out RPC response
				rf.nextIndex[ID], rf.matchIndex[ID] = myLastIndex+1, myLastIndex
			} else if rf.nextIndex[ID] > 1 {
				if replies[ID].LogLength != 0 && rf.nextIndex[ID] > replies[ID].LogLength {
					rf.nextIndex[ID] = replies[ID].LogLength
				} else {
					rf.nextIndex[ID]--
				}
			}
		}
	}

	// commit phase (leader)
	sortedMatchIndex := make([]int, len(rf.peers))
	copy(sortedMatchIndex, rf.matchIndex)
	sortedMatchIndex[rf.me] = 0
	sort.Ints(sortedMatchIndex)
	// maxCommit = maximal matchIndex s.t. strict majority ≥ it
	// e.g. len=7 [∞*, 2,4,5,6*,8*,8*] → 6; len=8: [∞*, 1,7,7,7*,8*,9*,9*] → 7
	maxCommit := Min(sortedMatchIndex[(len(sortedMatchIndex)+1)/2], len(rf.log)-1)
	for i := maxCommit; i > rf.commitIndex; i-- { // find maximal commitIndex
		if rf.log[i].Term == rf.currentTerm { // satisfying this
			rf.commitIndex = i
			go rf.applyEntries()
			break
		}
	}
	P("H-", rf.me, rf.currentTerm, beatNumber, "| next", rf.nextIndex, "| match", rf.matchIndex, "| commit", rf.commitIndex, "| log", rf.log, "| time", float64(time.Since(startTime).Nanoseconds())/1e9)
	rf.mu.Unlock()

	// recurse
	rf.heartbeat(beatNumber + 1)
}
