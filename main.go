package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/angelchiav/blog-aggregator-go/internal/commands"
	"github.com/angelchiav/blog-aggregator-go/internal/config"
	"github.com/angelchiav/blog-aggregator-go/internal/database"
)

func main() {

	// Arguments verification

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough arguments: need a command")
		os.Exit(1)
	}

	// Reading (Unmarshaling JSON)

	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// Starting PostgreSQL DB

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("error running the database: %v", err)
	}

	defer db.Close()

	// State Instance

	state := &commands.State{
		Cfg: &cfg,
		DB:  database.New(db),
	}

	cmdName := os.Args[1]
	cmdArgs := []string{}

	if len(os.Args) > 2 {
		cmdArgs = os.Args[2:]
	}
	cmd := commands.Command{
		Name: cmdName,
		Args: cmdArgs,
	}

	// Command registry
	reg := commands.Commands{}
	reg.Register("login", (*commands.State).HandlerLogin)
	reg.Register("register", (*commands.State).HandlerRegister)
	reg.Register("reset", (*commands.State).HandlerReset)
	reg.Register("users", (*commands.State).HandlerUsers)
	reg.Register("agg", (*commands.State).HandlerAgg)
	reg.Register("addfeed", commands.MiddlewareLoggedIn(commands.HandlerAddFeed))
	reg.Register("feeds", (*commands.State).HandlerGetFeed)
	reg.Register("follow", commands.MiddlewareLoggedIn(commands.HandlerFeedFollow))
	reg.Register("following", commands.MiddlewareLoggedIn(commands.HandlerFeedFollowing))
	reg.Register("unfollow", commands.MiddlewareLoggedIn(commands.HandlerFeedUnfollow))
	reg.Register("browse", commands.MiddlewareLoggedIn(commands.HandlerBrowse))

	if err := reg.Run(state, cmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
