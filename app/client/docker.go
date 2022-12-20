package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

type dockerImageManifest struct {
	Name     string
	Tag      string
	FsLayers []struct {
		BlobSum string
	}
}

type DockerClient struct {
	repo  string
	ref   string
	token string
}

func NewDockerClient(repo, ref, token string) *DockerClient {
	return &DockerClient{
		repo:  repo,
		ref:   ref,
		token: token,
	}
}

func (dockerClient *DockerClient) PullImageManifest() (*dockerImageManifest, error) {
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/%s/manifests/%s", dockerClient.repo, dockerClient.ref)

	resp, err := dockerClient.authenticatedRequest(url)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var manifest dockerImageManifest
	if err = json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (dockerClient *DockerClient) PullImageLayers(manifest *dockerImageManifest, tarDir string) ([]string, error) {
	var paths []string
	for _, layer := range manifest.FsLayers {
		blobSum := layer.BlobSum
		destPath := path.Join(tarDir, blobSum)

		err := dockerClient.PullLayer(destPath, blobSum)
		if err != nil {
			return nil, err
		}

		paths = append(paths, destPath)
	}
	return paths, nil
}

func (dockerClient *DockerClient) PullLayer(destPath, blobSum string) error {
	err := os.MkdirAll(path.Dir(destPath), 0750)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://registry.hub.docker.com/v2/%s/blobs/%s", dockerClient.repo, blobSum)
	resp, err := dockerClient.authenticatedRequest(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	io.Copy(f, resp.Body)
	return nil
}

func (dockerClient *DockerClient) authenticatedRequest(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+dockerClient.token)

	client := http.DefaultClient
	return client.Do(req)
}
