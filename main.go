package main

import (
	"fmt"
	"os"

	"github.com/angelchiav/blog-aggregator-go/internal/commands"
	"github.com/angelchiav/blog-aggregator-go/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough arguments: need a command")
		os.Exit(1)
	}

	cmdName := os.Args[1]
	cmdArgs := []string{}

	if len(os.Args) > 2 {
		cmdArgs = os.Args[2:]
	}

	cfg, err := config.Read()

	if err != nil {
		fmt.Fprintln(os.Stderr, "read config: ", err)
		os.Exit(1)
	}

	st := &commands.State{
		Cfg: &cfg,
	}

	reg := &commands.Commands{}
	reg.Register("login", commands.HandlerLogin)

	cmd := commands.Command{
		Name: cmdName,
		Args: cmdArgs,
	}
	if err := reg.Run(st, cmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
