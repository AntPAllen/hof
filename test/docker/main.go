package main

import (
	"fmt"
	"os"

	"github.com/hofstadter-io/hof/lib/docker"
)

func main() {
	fmt.Println("testing docker client/server compat")


	err := docker.InitDockerClient()
	if err != nil {
		fmt.Println("Error Initializing Client:", err)
		os.Exit(1)
	}

	client, server, err := docker.GetVersion()
	if err != nil {
		fmt.Println("Error getting versions:", err)
		os.Exit(1)
	}

	fmt.Println("client: ", client)
	fmt.Println("server:", server.Version, server.APIVersion, server.MinAPIVersion)

	img := "ghcr.io/hofstadter-io/fmt-black:v0.6.8-beta.11"

	err = docker.PullImage(img)
	if err != nil {
		fmt.Println("Error pulling image:", err)
		os.Exit(1)
	}

	images, err := docker.GetImages(img)
	if err != nil {
		fmt.Println("Error detailing image:", err)
		os.Exit(1)
	}

	for _, image := range images {
		fmt.Println(image.ID, image.RepoTags)
	}

}
