package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"bytes"
	"fmt"
	"labgob"
	"labrpc"
	"sync"
	"time"
)

// ApplyMsg ...
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//

// INIT

// ApplyMsg ...
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

// LogEntry ...
type LogEntry struct {
	Term int
	Data interface{}
}

const steadyElectionTimeout = 300 * time.Millisecond
const electionRandomisation = 200
const heartbeatTimeout = 50 * time.Millisecond
const voteTimeout = 200 * time.Millisecond
const exitDelay = 200 * time.Millisecond

var electionTimeout = 0 * time.Millisecond
var electionReset = make(chan bool)

// Raft ...
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	phase       string // leader or follower or candidate
	currentTerm int
	votedFor    int
	log         []LogEntry

	commitIndex int
	lastApplied int

	nextIndex  []int
	matchIndex []int

	applyCh chan ApplyMsg
}

// GetState ...
// return currentTerm, whether this server
// believes it is the leader, and length
// of log (= length of list - 1)
func (rf *Raft) GetState() (term int, isleader bool) {
	// Your code here (2A).
	rf.mu.Lock()
	term = rf.currentTerm
	isleader = rf.phase == "leader"
	rf.mu.Unlock()
	return
}

// PERSISTENCE

// persist ...
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
	P(rf.me, "saved")
}

// readPersist ...
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) == 0 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var log []LogEntry
	var votedFor int
	var currentTerm int
	if d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&log) != nil {
		fmt.Println("DECODE ERROR")
	} else {
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		rf.log = log
	}
}

// START, KILL, MAKE

// Start ...
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (index int, term int, isLeader bool) {
	// Your code here (2B).
	rf.mu.Lock()
	term = rf.currentTerm
	isLeader = rf.phase == "leader"
	if isLeader { // have lock so can reliably determine index
		rf.log = append(rf.log, LogEntry{term, command})
		index = len(rf.log) - 1 // I know Kappa
		rf.persist()
		P(rf.me, "input o", term, command)
	} else {
		P(rf.me, "input x", command)
	}
	rf.mu.Unlock()
	return
}

// Kill ...
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
	rf.mu.Lock()
	rf.phase = "exit"
	P(rf.me, "x")
	if rf.me == len(rf.peers)-1 {
		time.Sleep(exitDelay)
	}
	rf.mu.Unlock()
}

// Make ...
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.log = append(rf.log, LogEntry{})
	rf.applyCh = applyCh

	// Your initialization code here (2A, 2B, 2C).
	rf.votedFor = -1

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	P(rf.me, "loaded state | currentTerm=", rf.currentTerm, "| votedFor=", rf.votedFor, "| log=", rf.log)

	rf.phaseChange("follower", false, "init")
	return rf
}

// aux functions

// Min ...
func Min(a int, b int) int {
	if a <= b {
		return a
	}
	return b
}
