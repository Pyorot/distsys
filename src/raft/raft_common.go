package raft

func (rf *Raft) applyEntries() { // called whenever commitIndex is incremented
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
	if fromPhase == "exit" { // allows quick exit for testing
		P(rf.me, "xx")
		rf.mu.Unlock()
		return
	} else if fromPhase != toPhase {
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
	// syncs terms during RPC exchange, returns sgn(myTerm - otherTerm)
	rf.mu.Lock()
	myTerm = rf.currentTerm

	if myTerm < otherTerm { // update term, become follower
		rf.currentTerm = otherTerm
		rf.votedFor = -1 // resets upon term increase
		rf.persist()
		rf.mu.Unlock()
		rf.phaseChange("follower", false, "tS fail ("+RPCName+" "+senderReceiver+")")
		return -1, myTerm
	} else if myTerm == otherTerm {
		rf.mu.Unlock()
		return 0, myTerm
	} else {
		rf.mu.Unlock()
		return 1, myTerm
	}
}
