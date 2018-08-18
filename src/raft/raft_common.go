package raft

func (rf *Raft) applyEntries() {
	rf.mu.Lock()
	for rf.commitIndex > rf.lastApplied {
		rf.lastApplied++
		rf.applyCh <- ApplyMsg{CommandValid: true, Command: rf.log[rf.lastApplied].Data, CommandIndex: rf.lastApplied}
		P(rf.me, "output", rf.lastApplied, ":", rf.log[rf.lastApplied].Data)
	}
	rf.mu.Unlock()
}

func (rf *Raft) phaseChange(toPhase string, sync bool, reason string) (success bool) {
	initPhase := map[string]func(){
		"leader":    rf.initLeader,
		"follower":  rf.initFollower,
		"candidate": rf.initCandidate,
	}
	rf.mu.Lock()
	fromPhase := rf.phase
	if fromPhase != toPhase {
		rf.phase = toPhase
		P(rf.me, "→", toPhase, "|", reason)
		rf.mu.Unlock()
		if sync {
			initPhase[toPhase]()
		} else {
			go initPhase[toPhase]()
		}
		return true
	}
	P(rf.me, "-", toPhase, "|", reason)
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
		rf.phaseChange("follower", false, "tS fail ("+RPCName+" "+senderReceiver+")")
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
