package main

import (
	"encoding/json"
	"os"

	"github.com/natto1784/gitea-release-resource"
)

func main() {
	request := resource.NewCheckRequest()
	inputRequest(&request)

	gitea, err := resource.NewGiteaClient(request.Source)
	if err != nil {
		resource.Fatal("constructing gitea client", err)
	}

	command := resource.NewCheckCommand(gitea)
	response, err := command.Run(request)
	if err != nil {
		resource.Fatal("running command", err)
	}

	outputResponse(response)
}

func inputRequest(request *resource.CheckRequest) {
	if err := json.NewDecoder(os.Stdin).Decode(request); err != nil {
		resource.Fatal("reading request from stdin", err)
	}
}

func outputResponse(response []resource.Version) {
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		resource.Fatal("writing response to stdout", err)
	}
}
