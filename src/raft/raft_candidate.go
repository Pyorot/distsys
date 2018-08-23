package raft

import (
	"math"
	"strconv"
)

func (rf *Raft) initCandidate() {
	rf.callElection()
}

func (rf *Raft) callElection() {
	rf.mu.Lock()
	// candidate check
	if rf.phase != "candidate" {
		rf.mu.Unlock()
		P(rf.me, "x candidate | check 1")
		return
	}
	// set up vote
	rf.currentTerm++
	rf.votedFor = -1 // reset on each new term
	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateID:  rf.me,
		LastLogIndex: len(rf.log) - 1,
	}
	args.LastLogTerm = rf.log[args.LastLogIndex].Term
	rf.mu.Unlock()
	votes := make(chan bool, len(rf.peers)) // to collect votes
	voteCount := 0
	majority := int(math.Ceil(float64(len(rf.peers)) / 2))
	// request votes via RequestVote RPC
	replies := make([]RequestVoteReply, len(rf.peers))
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID == rf.me && (rf.votedFor == -1 || rf.votedFor == rf.me) {
			votes <- true
			rf.votedFor = rf.me
		} else {
			go rf.sendRequestVote(ID, &args, &replies[ID], votes)
		}
	}
	P(rf.me, "requested votes |", args.Term)
	// await votes
	for i := 0; i < len(rf.peers); i++ {
		select {
		case vote := <-votes:
			if vote {
				voteCount++
			}
		case <-electionReset: // triggered by newer leader
			rf.phaseChange("follower", false, "vote interrupt")
			P(rf.me, "x candidate")
			return
		}
		if voteCount >= majority {
			break
		}
	}
	P(rf.me, "received votes")
	rf.mu.Lock()

	// candidate check 2
	if rf.phase != "candidate" {
		rf.mu.Unlock()
		P(rf.me, "x candidate | check 2")
		return
	}
	rf.mu.Unlock()

	// continue
	if voteCount >= majority {
		// synchronous (want leader working asap)
		rf.phaseChange("leader", true, "elected ("+strconv.Itoa(voteCount)+" votes; term "+strconv.Itoa(rf.currentTerm)+")")
	} else {
		// async (low priority)
		rf.phaseChange("follower", false, "lost vote")
	}
	P(rf.me, "x candidate")
}
