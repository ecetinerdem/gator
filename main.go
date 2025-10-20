package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/ecetinerdem/gator/internal/config"
	"github.com/ecetinerdem/gator/internal/database"
	_ "github.com/lib/pq"
)

func main() {

	cfg, err := config.Read()

	if err != nil {
		log.Fatalf("couldn't set current user: %v", err)
	}

	state := config.State{
		Cfg: &cfg,
	}

	db, err := sql.Open("postgres", cfg.DBURL)

	if err != nil {
		log.Fatal("error for opening DB %w", err)
	}

	defer db.Close()

	dbQueries := database.New(db)

	state.DB = dbQueries

	cmds := &config.Commands{
		MapC: make(map[string]func(s *config.State, c config.Command) error),
	}

	cmds.Register("login", config.LoginHandler)
	cmds.Register("register", config.RegisterHandler)
	cmds.Register("reset", config.ResetHandler)
	cmds.Register("users", config.GetUsersHandler)
	cmds.Register("agg", config.AggHandler)
	//cmds.Register("addfeed", config.AddFeedHandler)
	cmds.Register("feeds", config.FeedsHandler)
	cmds.Register("addfeed", MiddlewareLoggedIn(config.AddFeedHandler))
	cmds.Register("follow", MiddlewareLoggedIn(config.FollowHandler))
	cmds.Register("following", MiddlewareLoggedIn(config.FollowingHandler))
	cmds.Register("unfollow", MiddlewareLoggedIn(config.UnfollowHandler))
	cmds.Register("browse", MiddlewareLoggedIn(config.BrowseHandler))

	if len(os.Args) < 2 {
		log.Fatal("no command provided")
	}

	cmdName := os.Args[1]

	cmdArgs := []string{}

	if len(os.Args) > 2 {
		cmdArgs = os.Args[2:]
	}

	cmd := config.Command{
		Name: cmdName,
		Args: cmdArgs,
	}

	err = cmds.Run(&state, cmd)
	if err != nil {
		log.Fatalf("error executing command: %v", err)
	}

}

func MiddlewareLoggedIn(handler func(s *config.State, cmd config.Command, user database.User) error) func(*config.State, config.Command) error {
	return func(s *config.State, cmd config.Command) error {
		user, err := s.DB.GetUser(context.Background(), s.Cfg.CurrentUserName)
		if err != nil {
			return fmt.Errorf("user not logged in or doesn't exist: %w", err)
		}
		return handler(s, cmd, user)
	}
}

//nvim ~/.gatorconfig.json
