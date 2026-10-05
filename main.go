package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("ghostdog-dungeon", flag.ContinueOnError)
	flags.SetOutput(stderr)
	seed := flags.Int64("seed", time.Now().UnixNano(), "random seed for dungeon generation")
	saveFile := flags.String("save-file", "", "fixed path used by save/load commands; otherwise saves use a timestamped name")
	loadFile := flags.String("load-file", "", "load this save file at startup")
	loadSave := flags.Bool("load", false, "load the path supplied with --save-file (legacy form)")
	recordsFile := flags.String("records-file", "", "challenge records path; defaults to ~/.ghost-dog-data.json")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: ghostdog-dungeon [--seed N] [--save-file path] [--load-file path]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Enter the dungeon, free the ghost dog, reach the tenth floor, and escape by finding the hidden way out beyond the Dread Lich.")
		fmt.Fprintln(stderr, "Commands: wasd/arrows move, qezx diagonals, . wait, </> stairs, p f b g u t o n r items, v fountain, y interact, i inventory, E equipment, B book, j challenges, c codex, m inspect, k search, :equip NUMBER, :drop NUMBER, :choose 1/2/3, S save, L load, Ctrl-C quit")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return errors.New("this game does not take positional arguments")
	}

	if *loadSave && *loadFile != "" {
		return errors.New("use either --load-file or --load, not both")
	}
	if *loadSave && *saveFile == "" {
		return errors.New("--load needs a path; use --load-file path or --load --save-file path")
	}
	var g *game
	var err error
	if *loadFile != "" || *loadSave {
		path := *loadFile
		if path == "" {
			path = *saveFile
		}
		g, err = loadGame(path)
		if err != nil {
			return err
		}
		g.addMessage("Loaded saved game.")
	} else {
		g = newGameWithSaveFile(*seed, *saveFile)
	}
	// Record storage belongs to the current user rather than a saved dungeon.
	if g.generationError != nil {
		return g.generationError
	}
	g.recordsFile = *recordsFile
	if _, err := g.challengeRecordsPath(); err != nil {
		return err
	}
	return g.loop(stdin, stdout)
}
