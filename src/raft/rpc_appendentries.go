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
	Term    int
	Success bool
}

// AppendEntries ...
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	otherTerm := args.Term
	outcome, myTerm := rf.termSync(otherTerm, "AppendEntries", "receiver")
	react := outcome <= 0 // tS react: should I react to this RPC at all?
	reply.Term = myTerm

	// payload
	if react {
		rf.mu.Lock()
		// 1. set success
		reply.Success = len(rf.log) > args.PrevLogIndex && rf.log[args.PrevLogIndex].Term == args.PrevLogTerm
		// 2. merge log
		if reply.Success {
			// NEXT SESH: delete rf.log[PrevLogIndex+1:] regardless of success, and auto-merge

			// 2a. resolve conflicts
			for i := 0; i < len(args.Entries) && i < len(rf.log)-1-args.PrevLogIndex; i++ {
				if args.Entries[i].Term != rf.log[args.PrevLogIndex+1+i].Term {
					rf.log = rf.log[:args.PrevLogIndex+1+i]
					break
				}
			}
			// 2b. add payload
			P("I:", rf.me, "; other:", args.LeaderID)
			P(rf.log)
			P(args.Entries)
			P(args.PrevLogIndex)
			rf.log = append(rf.log, args.Entries[len(rf.log)-(args.PrevLogIndex+1):]...)
		}
		// 3. update commitIndex
		if len(args.Entries) > 0 && args.LeaderCommit > rf.commitIndex {
			rf.commitIndex = Min(args.LeaderCommit, len(args.Entries)-1)
		}
		rf.mu.Unlock()
	}
	P("AppendEntries:", args.LeaderID, "<", rf.me, "|", otherTerm, "vs", myTerm)
}

// sendAppendEntries ...
func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) (ok bool) {
	P("AppendEntries:", rf.me, ">", server)
	ok = rf.peers[server].Call("Raft.AppendEntries", args, reply)
	// await reply here
	if !ok {
		P("AppendEntries:", rf.me, "?", server)
		return
	}
	otherTerm := reply.Term
	_, myTerm := rf.termSync(otherTerm, "AppendEntries", "sender")
	P("AppendEntries:", rf.me, "-", server, "|", myTerm, "vs", otherTerm)
	return
}
