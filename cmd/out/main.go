package main

import (
	"encoding/json"
	"os"

	"github.com/natto1784/gitea-release-resource"
)

func main() {
	if len(os.Args) < 2 {
		resource.Sayf("usage: %s <sources directory>\n", os.Args[0])
		os.Exit(1)
	}

	request := resource.NewOutRequest()
	inputRequest(&request)

	sourceDir := os.Args[1]

	gitea, err := resource.NewGiteaClient(request.Source)
	if err != nil {
		resource.Fatal("constructing gitea client", err)
	}

	command := resource.NewOutCommand(gitea, os.Stderr)
	response, err := command.Run(sourceDir, request)
	if err != nil {
		resource.Fatal("running command", err)
	}

	outputResponse(response)
}

func inputRequest(request *resource.OutRequest) {
	if err := json.NewDecoder(os.Stdin).Decode(request); err != nil {
		resource.Fatal("reading request from stdin", err)
	}
}

func outputResponse(response resource.OutResponse) {
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		resource.Fatal("writing response to stdout", err)
	}
}
