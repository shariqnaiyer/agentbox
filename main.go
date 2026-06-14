// Command box is agentbox's single binary: it acts as both the host-side
// supervisor (`box host`) and the client CLI (`box`, `box new`, `box ls`, ...).
package main

import box "github.com/shariqnaiyer/agentbox/cmd/box"

func main() {
	box.Execute()
}
