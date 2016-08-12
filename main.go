package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"golang.org/x/net/context"
	"gopkg.in/urfave/cli.v2"
	"os"
	"os/signal"
	"sync"
)

func actionRun(cc *cli.Context) error {
	token := cc.String("token")
	if token == "" {
		return cli.Exit("Missing bot token", 1)
	}

	session, err := discordgo.New(token)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	// Log connection state changes.
	session.AddHandler(func(_ *discordgo.Session, e *discordgo.Connect) {
		log.Info("Connected!")
	})
	session.AddHandler(func(_ *discordgo.Session, e *discordgo.Disconnect) {
		log.Warn("Disconnected!")
	})
	session.AddHandler(func(_ *discordgo.Session, e *discordgo.Resumed) {
		log.Info("Resumed!")
	})
	session.AddHandler(func(_ *discordgo.Session, e *discordgo.Ready) {
		log.WithFields(log.Fields{
			"protocol": e.Version,
			"username": fmt.Sprintf("%s#%s", e.User.Username, e.User.Discriminator),
		}).Info("Ready!")
	})

	// Run the Responder and the Player in goroutines.
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	responder := Responder{Session: session}
	wg.Add(1)
	go func() {
		log.Info("Responder: Initializing")
		responder.Run(ctx)
		log.Info("Responder: Terminated")
		wg.Done()
	}()

	player := Player{Session: session}
	wg.Add(1)
	go func() {
		log.Info("Player: Initializing")
		player.Run(ctx)
		log.Info("Player: Terminated")
		wg.Done()
	}()

	// Connect to Discord.
	if err := session.Open(); err != nil {
		log.WithError(err).Error("Couldn't connect to Discord!")
		return err
	}

	// Wait for a signal before exiting.
	quit := make(chan os.Signal)
	signal.Notify(quit)
	<-quit

	// Shut down subsystems, wait for them to finish.
	cancel()
	wg.Wait()

	return nil
}

func main() {
	app := cli.App{}
	app.Name = "hiqty"
	app.Usage = "A high quality Discord music bot"
	app.HideVersion = true
	app.Commands = []*cli.Command{
		&cli.Command{
			Name:   "run",
			Usage:  "Runs the bot interface + player",
			Action: actionRun,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "token",
					Aliases: []string{"t"},
					Usage:   "Discord token",
					EnvVars: []string{"HIQTY_BOT_TOKEN"},
				},
			},
		},
	}
	app.Before = func(cc *cli.Context) error {
		if err := godotenv.Load(); err != nil {
			return cli.Exit(err.Error(), 1)
		}

		return nil
	}
	if app.Run(os.Args) != nil {
		os.Exit(1)
	}
}
