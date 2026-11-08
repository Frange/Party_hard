package main

// The slow motion of the last kill delays the end of a party by up to a second; the old tests expect it at once.
// They run with it off; the slow motion tests switch it on and restore this with t.Cleanup.
func init() { slowmoEnabled = false }
