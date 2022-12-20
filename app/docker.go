package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

type DockerAPI struct {
	repo      string
	ref       string
	authToken string
}

func NewDockerAPI(image string) *DockerAPI {
	parts := strings.Split(image, ":")
	repo := parts[0]
	ref := "latest"
	if len(parts) == 2 {
		ref = parts[1]
	}
	if !strings.Contains(repo, "/") {
		repo = "library/" + repo
	}
	return &DockerAPI{
		ref:  ref,
		repo: repo,
	}
}

type dockerAuthResp struct {
	Token string
}

func (docker *DockerAPI) Auth() error {
	authUrl := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", docker.repo)
	resp, err := http.Get(authUrl)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var authResp dockerAuthResp
	err = json.Unmarshal(body, &authResp)
	if err != nil {
		return err
	}
	docker.authToken = authResp.Token
	return nil
}

type imageManifest struct {
	Name     string
	Tag      string
	FsLayers []struct {
		BlobSum string
	}
}

func (docker *DockerAPI) GetManifest() (*imageManifest, error) {
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/%s/manifests/%s", docker.repo, docker.ref)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if err = docker.addAuthHeader(req); err != nil {
		return nil, err
	}
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var manifest imageManifest
	if err = json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (docker *DockerAPI) GetBlobResp(blob string) (*http.Response, error) {
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/%s/blobs/%s", docker.repo, blob)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if err = docker.addAuthHeader(req); err != nil {
		return nil, err
	}
	client := http.DefaultClient
	return client.Do(req)
}

func (docker *DockerAPI) DownloadImage() ([]string, error) {
	manifest, err := docker.GetManifest()
	fmt.Println("Getting manifest")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, layer := range manifest.FsLayers {
		blob := layer.BlobSum
		path, err := ensureLayerDownloaded(docker, blob)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func (docker *DockerAPI) addAuthHeader(req *http.Request) error {
	req.Header.Add("Authorization", "Bearer "+docker.authToken)
	return nil
}

func ensureLayerDownloaded(docker *DockerAPI, blobsum string) (string, error) {
	destPath := path.Join(os.TempDir(), "mydocker", "layers", blobsum)
	_, err := os.Stat(destPath)
	fmt.Printf("Checking %s\n", destPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err = downloadLayer(docker, destPath, blobsum); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return destPath, nil
}

func downloadLayer(docker *DockerAPI, destPath, blobsum string) error {
	fmt.Printf("Downloading layer '%s'\n", blobsum)
	if err := os.MkdirAll(path.Dir(destPath), 0750); err != nil {
		return err
	}
	resp, err := docker.GetBlobResp(blobsum)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}
