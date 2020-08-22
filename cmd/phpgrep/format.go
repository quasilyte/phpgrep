package main

type formatData struct {
	// Filename is a match containing file name.
	// Could be relative (default) or absolute if -abs
	// command-line flag was used.
	Filename string

	// Line is a line number where the match started.
	Line int

	// Match is an entire match string.
	Match string
}
