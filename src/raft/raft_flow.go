package raft

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

func (rf *Raft) initLeader() {
	rf.mu.Lock()
	rf.nextIndex = make([]int, len(rf.peers))
	for i := range rf.nextIndex {
		rf.nextIndex[i] = len(rf.log)
	}
	rf.matchIndex = make([]int, len(rf.peers))
	rf.mu.Unlock()
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
	// leader check
	if rf.phase != "leader" {
		P(rf.me, "x leader")
		rf.mu.Unlock()
		return
	}
	// send heartbeats
	replies := make([]AppendEntriesReply, len(rf.peers))
	myLastIndex := len(rf.log) + 1
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID != rf.me {
			nextIndex := rf.nextIndex[ID]
			args := AppendEntriesArgs{
				Term:         rf.currentTerm,
				LeaderID:     rf.me,
				LeaderCommit: rf.commitIndex,
				PrevLogIndex: nextIndex - 1,
				PrevLogTerm:  rf.log[nextIndex-1].Term, // nextIndex ≥ 1
			}
			if nextIndex < len(rf.log) {
				args.Entries = rf.log[nextIndex:]
			}
			go rf.sendAppendEntries(ID, &args, &replies[ID])
		}
	}
	rf.mu.Unlock()

	// heartbeat timeout
	time.Sleep(heartbeatTimeout)

	// adjusting follower indices depending on responses
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID != rf.me {
			if replies[ID].Success {
				rf.nextIndex[ID], rf.matchIndex[ID] = myLastIndex+1, myLastIndex+1
			} else {
				rf.nextIndex[ID]--
			}
		}
	}

	// commit phase (leader)
	rf.mu.Lock()
	sortedMatchIndex := make([]int, len(rf.peers))
	copy(sortedMatchIndex, rf.matchIndex)
	sortedMatchIndex[rf.me] = 0
	sort.Ints(sortedMatchIndex)
	maxCommit := sortedMatchIndex[(len(sortedMatchIndex)+1)/2]
	// 7: [∞*, 2,4,5,6*,8*,8*] = 6; 8: [∞*, 1,7,7,7*,8*,9*,9*] = 7
	for i := maxCommit; i > rf.commitIndex; i-- {
		if rf.log[i].Term == rf.currentTerm {
			rf.commitIndex = i
			break
		}
	}
	rf.mu.Unlock()

	// recurse
	rf.heartbeat()
}

func (rf *Raft) awaitElection() {
	rf.mu.Lock()
	// follower check
	if rf.phase != "follower" {
		P(rf.me, "x follower")
		rf.mu.Unlock()
		return
	}
	rf.mu.Unlock()
	// continue after election timeout (recurse or > candidate)
	timeout := (time.Duration(rand.Intn(electionRandomisation)) * time.Millisecond) + electionTimeout
	select {
	case <-electionReset:
		P(rf.me, "- follower | reset")
		go rf.awaitElection()
	case <-time.After(timeout):
		P(rf.me, "→ candidate | timeout")
		rf.phaseChange("candidate", false)
		P(rf.me, "x follower")
	}
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
	term := rf.currentTerm
	rf.mu.Unlock()
	votes := make(chan bool, len(rf.peers))
	voteCount := 0
	majority := int(math.Ceil(float64(len(rf.peers)) / 2))
	// request votes via RequestVote RPC
	replies := make([]RequestVoteReply, len(rf.peers))
	for ID := 0; ID < len(rf.peers); ID++ {
		if ID == rf.me {
			votes <- true
		} else {
			args := RequestVoteArgs{Term: term, CandidateID: rf.me}
			go rf.sendRequestVote(ID, &args, &replies[ID], votes)
		}
	}
	P(rf.me, "requested votes |", term)
	// await votes
	for i := 0; i < len(rf.peers); i++ {
		select {
		case vote := <-votes:
			if vote {
				voteCount++
			}
		case <-electionReset:
			P(rf.me, "→ follower | vote interrupt")
			rf.phaseChange("follower", false)
			P(rf.me, "x candidate")
			return
		}
		// case 3: explicit election timeout?
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
		P(rf.me, "→ leader | elected:", voteCount, "votes")
		rf.phaseChange("leader", true)
		P(rf.me, "x candidate")
	} else {
		P(rf.me, "- candidate | lost vote")
		// randomised timeout?
		go rf.callElection()
	}
}

func (rf *Raft) applyEntries() {
	rf.mu.Lock()
	if rf.commitIndex > rf.lastApplied {
		rf.lastApplied++
		rf.applyCh <- ApplyMsg{CommandValid: true, Command: rf.log[rf.lastApplied].Data, CommandIndex: rf.lastApplied}
	}
	rf.mu.Unlock()
	go rf.applyEntries()
}

func (rf *Raft) phaseChange(toPhase string, sync bool) (success bool) {
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

func (rf *Raft) termSync(otherTerm int, RPCName string, senderReceiver string) (outcome int, myTerm int) {
	rf.mu.Lock()
	myTerm = rf.currentTerm
	if myTerm < otherTerm {
		rf.currentTerm = otherTerm
		rf.votedFor = -1
		rf.mu.Unlock()
		P(rf.me, "→ follower | term sync fail:", RPCName, senderReceiver)
		rf.phaseChange("follower", false)
		return -1, myTerm
	} else if myTerm == otherTerm {
		rf.mu.Unlock()
		if senderReceiver == "receiver" {
			go func() { electionReset <- true }()
		}
		return 0, myTerm
	} else {
		rf.mu.Unlock()
		return 1, myTerm
	}
}
