package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func parseImage(image string) (string, string) {
	parts := strings.Split(image, ":")
	repo := "library/" + parts[0]
	ref := "latest"
	if len(parts) == 2 {
		ref = parts[1]
	}
	return repo, ref
}

func authenticateWithDockerRegistry(repo string) (string, error) {
	authUrl := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", repo)

	resp, err := http.Get(authUrl)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var authResp dockerAuthResp
	err = json.Unmarshal(body, &authResp)
	if err != nil {
		return "", err
	}

	return authResp.Token, nil
}
