package raft

// APPLYENTRIES RPC

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
	go func() { electionReset <- true }()
	// rpc term sync
	if rf.currentTerm < args.Term {
		rf.currentTerm = args.Term
		rf.votedFor = -1
		// go rf.initFollower()
	}
	// payload
	if rf.currentTerm <= args.Term {
		reply.Success = true
	}
	reply.Term = rf.currentTerm
	P("AppendEntries:", args.LeaderID, "<", rf.me, "|", reply.Success)
	go rf.initFollower()
	rf.mu.Unlock()
}

// sendAppendEntries ...
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	P("AppendEntries:", args.LeaderID, ">", server)
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	// rpc term sync
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.currentTerm < reply.Term {
		rf.currentTerm = reply.Term
		rf.votedFor = -1
		go rf.initFollower()
	}
	return ok
}
