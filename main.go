package main

import (
	"borbAggregatorFRFR/internal/config"
	"borbAggregatorFRFR/internal/database"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

const dbUrl = `postgres://postgres:postgres@localhost:5432/gator`

func main() {

	conf := config.Read()
	var stateVar config.State
	stateVar.Conf = &conf
	db, err := sql.Open("postgres", dbUrl)
	stateVar.Db = database.New(db)

	var cmds config.Commands
	cmds.CommandFunc = make(map[string]func(*config.State, config.Command) error)
	cmds.Register("login", config.HandlerLogin)
	cmds.Register("register", config.HandlerRegister)
	cmds.Register("reset", config.HandlerReset)
	cmds.Register("users", config.HandlerGetUsers)
	cmds.Register("agg", config.HandlerScrapeFeeds)
	cmds.Register("addfeed", config.MiddlewareLoggedIn(config.HandlerAddFeed))
	cmds.Register("feeds", config.HandlerFeeds)
	cmds.Register("follow", config.MiddlewareLoggedIn(config.HandlerFollow))
	cmds.Register("following", config.MiddlewareLoggedIn(config.HandlerFollowing))
	cmds.Register("unfollow", config.MiddlewareLoggedIn(config.HandlerUnfollow))
	cmds.Register("browse", config.MiddlewareLoggedIn(config.HandlerBrowse))

	userInput := os.Args
	if len(userInput) < 2 {
		fmt.Println("not enough arguments")
		os.Exit(1)
	}

	var cmd config.Command
	cmd.Name = userInput[1]
	if len(userInput) > 2 {
		cmd.Arguments = userInput[2:]
	}

	err = cmds.Run(&stateVar, cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
