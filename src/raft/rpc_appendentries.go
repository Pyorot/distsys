package raft

// APPENDENTRIES RPC

// AppendEntriesArgs ...
type AppendEntriesArgs struct {
	Term         int
	LeaderID     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []string
	LeaderCommit int
}

// AppendEntriesReply ...
type AppendEntriesReply struct {
	Term    int
	Success bool
}

// AppendEntries ...
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	// rpc term sync
	if rf.currentTerm < args.Term {
		rf.currentTerm = args.Term
		rf.votedFor = -1
		rf.mu.Unlock()
		P(rf.me, "follower | term sync fail (AppendEntries receiver)")
		rf.phaseChange("follower", false)
		rf.mu.Lock()
	}
	if rf.currentTerm <= args.Term {
		go func() { electionReset <- true }()
		reply.Success = true
	}
	// payload
	reply.Term = rf.currentTerm
	P("AppendEntries:", args.LeaderID, "<", rf.me, "|", rf.currentTerm, "|", reply.Success)
	rf.mu.Unlock()
}

// sendAppendEntries ...
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	P("AppendEntries:", args.LeaderID, ">", server)
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	// rpc term sync
	rf.mu.Lock()
	if rf.currentTerm < reply.Term {
		rf.currentTerm = reply.Term
		rf.votedFor = -1
		rf.mu.Unlock()
		P(rf.me, "follower | term sync fail (AppendEntries sender)")
		rf.phaseChange("follower", false)
	} else {
		rf.mu.Unlock()
	}
	return ok
}
