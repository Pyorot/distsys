package raft

import "math/rand"
import "time"

func (rf *Raft) initLeader() {
	rf.mu.Lock()
	if rf.phase == "candidate" {
		rf.phase = "leader"
		P(rf.me, rf.phase)
		rf.mu.Unlock()
		rf.heartbeat()
	} else {
		rf.mu.Unlock()
		return
	}
}

func (rf *Raft) heartbeat() {
	rf.mu.Lock()
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID != rf.me {
			args := AppendEntriesArgs{Term: rf.currentTerm, LeaderID: rf.me}
			go rf.sendAppendEntries(ID, &args, &AppendEntriesReply{})
		}
	}
	rf.mu.Unlock()
	time.Sleep(heartbeatTimeout)
	var phase string
	rf.mu.Lock()
	phase = rf.phase
	rf.mu.Unlock()
	if phase == "leader" {
		rf.heartbeat()
	}
}

func (rf *Raft) initFollower() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.phase != "follower" { // should never re-init follower
		rf.phase = "follower"
		P(rf.me, rf.phase)
		go rf.awaitElection()
	}
}

func (rf *Raft) awaitElection() {
	rf.mu.Lock()
	if rf.phase != "follower" {
		return
	}
	rf.mu.Unlock()
	timeout := (time.Duration(rand.Intn(electionRandomisation)) * time.Millisecond) + electionTimeout
	select {
	case <-electionReset:
		P(rf.me, "reset")
		go rf.awaitElection()
	case <-time.After(timeout):
		P(rf.me, "timeout")
		go rf.initCandidate()
	}
}

func (rf *Raft) initCandidate() {
	rf.mu.Lock()
	rf.phase = "candidate"
	P(rf.me, rf.phase)
	rf.currentTerm++
	votes := make(chan bool, len(rf.peers))
	voteCount := 0
	// request votes via RequestVote RPC
	replies := make([]RequestVoteReply, len(rf.peers)) // pointers?
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID == rf.me {
			votes <- true
		} else {
			args := RequestVoteArgs{Term: rf.currentTerm, CandidateID: rf.me}
			go rf.sendRequestVote(ID, &args, &replies[ID], votes)
		}
	}
	rf.mu.Unlock()
	// await votes
	for i := 0; i < len(rf.peers); i++ {
		select {
		case vote := <-votes:
			if vote {
				voteCount++
			}
		case <-electionReset:
			rf.initFollower()
			return
		}
	}
	rf.mu.Lock()
	if rf.phase == "candidate" && voteCount >= len(rf.peers)/2 {
		go rf.initLeader()
	} else {
		go rf.initCandidate()
	}
	rf.mu.Unlock()
}
