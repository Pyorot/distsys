# Which files are which (project 2: Raft)
(Currently relevant files only)

## ./src/raft - the project code
* raft.go - interface functions, like the initiator `Make`, the data inputter `Start`
* raft_[common/leader/follower/candidate].go - raft internals, functions running heartbeats, elections, etc.
* raft_appendentries.go - AppendEntries RPC send/receive structs + functions
* raft_requestvote.go - RequestVote RPC send/receive structs + functions

## ./scrap - my text notes and sketches
* 2A - my scrap for part A of Raft (now implemented)
* 2B - my current scrap for part B of Raft
* raft code so far.txt - a summary of my code as of finishing part A
* raft index.txt - which functions+structs are which in the raft*.go source code files
* raft interface.txt - my notes on raft.go (interface functions)
* raft sketch.txt - my old sketch of the whole of raft prior to part A and B
