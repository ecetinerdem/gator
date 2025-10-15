package main

import (
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
	fmt.Printf("Read config again: %+v\n", cfg)

	state := config.State{
		Cfg: &cfg,
	}

	db, err := sql.Open("postgres", cfg.DBURL)

	if err != nil {
		fmt.Errorf("error for opening DB")
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

//nvim ~/.gatorconfig.json
