package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joejulian/go-init/pkg/sysinit"
)

type environment []string

var (
	versionString = "undefined"
	env           environment
	termTimeout   time.Duration
)

func (e *environment) String() string {
	return strings.Join(*e, " ")
}

func (e *environment) Set(value string) error {
	*e = environment(append(*e, value))
	return nil
}

func main() {
	var preStartCmd string
	var mainCmd string
	var postStopCmd string
	var version bool

	flag.StringVar(&preStartCmd, "pre", "", "Pre-start command")
	flag.StringVar(&mainCmd, "main", "", "Main command")
	flag.StringVar(&postStopCmd, "post", "", "Post-stop command")
	flag.DurationVar(&termTimeout, "term_timeout", 0, "Wait for this long before shutting down the main command")
	flag.Var(&env, "env", "Environment variable NAME=VALUE (can be used multiple times)")
	flag.BoolVar(&version, "version", false, "Display go-init version")
	flag.Parse()

	if version {
		fmt.Println(versionString)
		os.Exit(0)
	}

	if mainCmd == "" {
		log.Fatal("[go-init] No main command defined, exiting")
	}

	// Routine to reap zombies (it's the job of init)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go sysinit.RemoveZombies(ctx, &wg)

	// Launch pre-start command
	if preStartCmd == "" {
		log.Println("[go-init] No pre-start command defined, skip")
	} else {
		log.Printf("[go-init] Pre-start command launched : %s\n", preStartCmd)
		err := sysinit.Run(preStartCmd)
		if err != nil {
			log.Println("[go-init] Pre-start command failed")
			log.Printf("[go-init] %s\n", err)
			sysinit.CleanQuit(cancel, &wg, 1)
		} else {
			log.Printf("[go-init] Pre-start command exited")
		}
	}

	// Launch main command
	var mainRC int
	log.Printf("[go-init] Main command launched : %s\n", mainCmd)
	err := sysinit.Run(mainCmd)
	if err != nil {
		log.Println("[go-init] Main command failed")
		log.Printf("[go-init] %s\n", err)
		mainRC = 1
	} else {
		log.Printf("[go-init] Main command exited")
	}

	// Launch post-stop command
	if postStopCmd == "" {
		log.Println("[go-init] No post-stop command defined, skip")
	} else {
		log.Printf("[go-init] Post-stop command launched : %s\n", postStopCmd)
		err := sysinit.Run(postStopCmd)
		if err != nil {
			log.Println("[go-init] Post-stop command failed")
			log.Printf("[go-init] %s\n", err)
			sysinit.CleanQuit(cancel, &wg, 1)
		} else {
			log.Printf("[go-init] Post-stop command exited")
		}
	}

	// Wait removeZombies goroutine
	sysinit.CleanQuit(cancel, &wg, mainRC)
}
