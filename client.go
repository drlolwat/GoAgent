package main

// todo: replace empty struct with any data we want to track about each account
// e.g.
//
//type Client struct {
//	id        int
//	status    string
//	startedAt time.Time
//}
//
//var clients = map[int]*Client{}

var clients = map[int]*struct{}{}
