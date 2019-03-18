package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
)

func handleRunContainer(a *api, img, cmd string) error {
	if img == "" {
		return fmt.Errorf("image not provided")
	}

	if cmd == "" {
		return a.RunContainerBackground(img)
	}

	// TODO: handle command splitting in a more sane way
	out, err := a.RunContainerCmd(img, strings.Split(cmd, " "))
	if err != nil {
		return err
	}
	fmt.Println(out)

	return nil
}

func handleListContainers(a *api) error {
	cs, err := a.ListContainers()
	if err != nil {
		return err
	}

	for _, c := range cs {
		fmt.Println(c.ID)
	}

	return nil
}

func main() {
	mode := flag.String("mode", "list", "run, stop, or list")
	img := flag.String("image", "", "Container image, e.g. docker.io/library/alpine")
	cmd := flag.String("cmd", "", "Command to be ran in the container. Optional")
	id := flag.String("id", "", "Container ID")
	flag.Parse()

	a, err := newAPI()
	if err != nil {
		log.Fatalf("new API: %s", err)
	}

	switch *mode {
	case "run":
		if err := handleRunContainer(a, *img, *cmd); err != nil {
			log.Fatalf("run container: %s", err)
		}
	case "stop":
		if err := a.StopContainer(*id); err != nil {
			log.Fatalf("stop container: %s", err)
		}
	case "list":
		if err := handleListContainers(a); err != nil {
			log.Fatalf("list containers: %s", err)
		}
	}
}
