package raft

import (
	"math"
	"math/rand"
	"time"
)

func (rf *Raft) initLeader() {
	rf.heartbeat()
}
func (rf *Raft) initFollower() {
	rf.awaitElection()
}
func (rf *Raft) initCandidate() {
	rf.callElection()
}

func (rf *Raft) heartbeat() {
	rf.mu.Lock()
	if rf.phase != "leader" {
		P(rf.me, "[exit] leader")
		rf.mu.Unlock()
		return
	}
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID != rf.me {
			args := AppendEntriesArgs{Term: rf.currentTerm, LeaderID: rf.me}
			go rf.sendAppendEntries(ID, &args, &AppendEntriesReply{})
		}
	}
	rf.mu.Unlock()
	time.Sleep(heartbeatTimeout)
	rf.heartbeat()
}

func (rf *Raft) awaitElection() {
	rf.mu.Lock()
	if rf.phase != "follower" {
		P(rf.me, "[exit] follower")
		rf.mu.Unlock()
		return
	}
	rf.mu.Unlock()
	timeout := (time.Duration(rand.Intn(electionRandomisation)) * time.Millisecond) + electionTimeout
	select {
	case <-electionReset:
		P(rf.me, "(follower) | reset")
		go rf.awaitElection()
	case <-time.After(timeout):
		P(rf.me, "candidate | timeout")
		rf.phaseChange("candidate", false)
		P(rf.me, "[exit] follower")
	}
}

func (rf *Raft) callElection() {
	var term int
	rf.mu.Lock()
	if rf.phase != "candidate" {
		rf.mu.Unlock()
		P(rf.me, "[exit] candidate")
		return
	}
	rf.currentTerm++
	term = rf.currentTerm
	rf.mu.Unlock()
	votes := make(chan bool, len(rf.peers))
	voteCount := 0
	majority := int(math.Ceil(float64(len(rf.peers)) / 2))
	// request votes via RequestVote RPC
	replies := make([]RequestVoteReply, len(rf.peers)) // pointers?
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID == rf.me {
			votes <- true
		} else {
			args := RequestVoteArgs{Term: term, CandidateID: rf.me}
			go rf.sendRequestVote(ID, &args, &replies[ID], votes)
		}
	}
	P(rf.me, "requested votes")
	// await votes
	for i := 0; i < len(rf.peers); i++ {
		if voteCount >= majority {
			break
		}
		select {
		case vote := <-votes:
			if vote {
				voteCount++
			}
		case <-electionReset:
			P(rf.me, "follower | vote interrupt")
			rf.phaseChange("follower", false)
			P(rf.me, "[exit] candidate")
			return
		}
	}
	P(rf.me, "received votes")
	// vote count and decision
	rf.mu.Lock()
	if rf.phase == "candidate" && voteCount >= majority {
		P(rf.me, "leader | elected:", voteCount, "votes")
		rf.mu.Unlock()
		P(rf.me, "[exit] candidate")
		rf.phaseChange("leader", true)
	} else {
		if rf.phase == "candidate" {
			P(rf.me, "(candid) | lost vote")
		} else {
			P(rf.me, "(candid) | lost candidacy")
		}
		rf.mu.Unlock()
		go rf.callElection()
	}
}

func (rf *Raft) phaseChange(toPhase string, sync bool) bool {
	initPhase := map[string]func(){
		"leader":    rf.initLeader,
		"follower":  rf.initFollower,
		"candidate": rf.initCandidate,
	}
	rf.mu.Lock()
	fromPhase := rf.phase
	if fromPhase != toPhase {
		rf.phase = toPhase
		rf.mu.Unlock()
		if sync {
			initPhase[toPhase]()
		} else {
			go initPhase[toPhase]()
		}
		return true
	}
	rf.mu.Unlock()
	return false
}
