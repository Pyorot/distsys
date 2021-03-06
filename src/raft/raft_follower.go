package raft

import (
	"math/rand"
	"time"
)

func (rf *Raft) initFollower() {
	rf.awaitElection()
}

func (rf *Raft) awaitElection() {
	rf.mu.Lock()
	// follower check
	if rf.phase != "follower" {
		P(rf.me, "x follower")
		rf.mu.Unlock()
		return
	}
	// continue after election timeout (recurse or > candidate)
	timeout := (time.Duration(rand.Intn(electionRandomisation)) * time.Millisecond) + electionTimeout
	electionTimeout = steadyElectionTimeout // = 0 at first run
	rf.mu.Unlock()
	select {
	case <-electionReset: // triggered by newer leader
		P(rf.me, "- follower | reset")
		go rf.awaitElection()
	case <-time.After(timeout):
		rf.phaseChange("candidate", false, "timeout")
		P(rf.me, "x follower")
	}
}
