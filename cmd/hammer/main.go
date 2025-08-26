package main

import (
	"fmt"
	"github.com/leninmehedy/solo-chaos/cmd/hammer/commands"
	"os"
)

// hammer tx crypto --nodes "node1,node2" --mirror-node mirror:80 --bots 100 --duration 10s --tps 10 // Total TPS = 100 * 10 = 1000
// hammer tx crypto --config local.yml --bots 1000 --duration 10s --tps 100

func main() {
	err := commands.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
