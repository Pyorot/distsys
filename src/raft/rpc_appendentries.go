package raft

// APPENDENTRIES RPC

// AppendEntriesArgs ...
type AppendEntriesArgs struct {
	Term         int
	LeaderID     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

// AppendEntriesReply ...
type AppendEntriesReply struct {
	Term      int
	Success   bool
	NextIndex int // set only if Success=false
}

// AppendEntries ...
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// term sync
	otherTerm := args.Term
	outcome, myTerm := rf.termSync(otherTerm, "AppendEntries", "receiver")
	react := outcome <= 0 // tS react: should I react to this RPC at all?
	reply.Term = myTerm

	// payload
	if react {
		go func() { electionReset <- true }()
		rf.mu.Lock()

		// 1. set Success, LogLength
		if args.PrevLogIndex < len(rf.log) {
			reply.Success = rf.log[args.PrevLogIndex].Term == args.PrevLogTerm

			if reply.Success {
				// 2. merge log
				rf.log = append(rf.log[:args.PrevLogIndex+1], args.Entries...)

				// 3. update commitIndex
				newCommitIndex := Min(args.LeaderCommit, len(rf.log)-1)
				if newCommitIndex > rf.commitIndex {
					rf.commitIndex = newCommitIndex // has def increased
					go rf.applyEntries()
				}

				rf.persist()
			} else { // PrevLogIndex in bounds but term clash
				conflictTerm := rf.log[args.PrevLogIndex].Term
				// ConflictIndex is minimal entry with same term
				// s.t. every entry from it to PrevLogIndex has same term
				// – speculating on reverting all entries with that leader
				reply.NextIndex = args.PrevLogIndex
				for i := args.PrevLogIndex - 1; i >= 0; i-- {
					if rf.log[i].Term != conflictTerm {
						reply.NextIndex = i + 1
						break
					}
				}
			}
		} else { // PrevLogIndex out-of-bounds
			reply.NextIndex = len(rf.log)
		}

		rf.mu.Unlock()
	}
	P("AppendEntries:", args.LeaderID, "<", rf.me, "|", otherTerm, "vs", myTerm, "| react", react, "| success", reply.Success)
}

// sendAppendEntries ...
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) (ok bool) {
	P("AppendEntries:", rf.me, ">", server)
	ok = rf.peers[server].Call("Raft.AppendEntries", args, reply)

	// await reply here
	if !ok { // i.e. RPC timeout
		P("AppendEntries:", rf.me, "?", server)
		return
	}

	// term sync
	otherTerm := reply.Term
	_, myTerm := rf.termSync(otherTerm, "AppendEntries", "sender")
	P("AppendEntries:", rf.me, "-", server, "|", myTerm, "vs", otherTerm)
	return
}
