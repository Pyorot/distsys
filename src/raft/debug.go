package raft

import "fmt"

// Debug ...
const Debug = 0

// P (DPrintf)
func P(a ...interface{}) (n int, err error) {
	if Debug > 0 {
		fmt.Println(a...)
	}
	return
}
