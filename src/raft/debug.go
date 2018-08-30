package raft

import "fmt"

// Debug ...
var Debug = 1

// P (DPrintf)
func P(a ...interface{}) (n int, err error) {
	if Debug > 0 {
		fmt.Println(a...)
	}
	return
}

// Debuq ...
var Debuq = 1

// Q (DPrintf)
func Q(a ...interface{}) (n int, err error) {
	if Debuq > 0 {
		fmt.Println(a...)
	}
	return
}
