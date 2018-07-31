package raft

// REQUESTVOTE RPC

// RequestVoteArgs ...
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateID  int
	LastLogIndex int
	LastLogTerm  int
}

// RequestVoteReply ...
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

// RequestVote ...
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	otherTerm := args.Term
	outcome, myTerm := rf.termSync(otherTerm, "RequestVote", "receiver")
	react := outcome <= 0 // tS react: should I react to this RPC at all?
	reply.Term = myTerm

	// payload
	if react {
		rf.mu.Lock()
		if rf.votedFor == -1 || rf.votedFor == args.CandidateID {
			// vote should also depend on log updated-ness (tbd in Part B)
			reply.VoteGranted = true
			rf.votedFor = args.CandidateID
		}
		rf.mu.Unlock()
	}
	P("RequestVote:", args.CandidateID, "<", rf.me, "|", otherTerm, "vs", myTerm, "| vote:", reply.VoteGranted)
}

// sendRequestVote ...
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply, doneCh chan bool) (ok bool) {
	P("RequestVote:", rf.me, ">", server)
	ok = rf.peers[server].Call("Raft.RequestVote", args, reply)
	// await reply here
	if !ok {
		doneCh <- false
		P("RequestVote:", rf.me, "?", server)
		return
	}
	otherTerm := reply.Term
	_, myTerm := rf.termSync(otherTerm, "RequestVote", "sender")
	doneCh <- reply.VoteGranted // could deadlock
	P("RequestVote:", rf.me, "-", server, "|", myTerm, "vs", otherTerm)
	return
}