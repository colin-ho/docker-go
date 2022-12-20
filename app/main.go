package main

import (
	"codecrafters-docker-go/app/client"
	"codecrafters-docker-go/app/utils"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	if len(os.Args) < 4 {
		fmt.Println("Not enough arguments, expecting 4")
		os.Exit(1)
	}

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]
	image := os.Args[2]

	chrootDir, err := os.MkdirTemp("", "chroot")
	if err != nil {
		fmt.Printf("error creating chroot dir: %v", err)
		os.Exit(1)
	}

	tarDir, err := os.MkdirTemp("", "tarDir")
	if err != nil {
		fmt.Printf("error creating temp folder for tar files: %v", err)
		os.Exit(1)
	}

	defer cleanUp(chrootDir, tarDir)

	err = utils.CopyExecutableIntoDir(chrootDir, command)
	if err != nil {
		fmt.Printf("error copying executable into chroot dir: %v", err)
		os.Exit(1)
	}

	paths := fetchDockerImageIntoDir(image, tarDir)

	err = utils.ExtractTarsToDir(chrootDir, paths)
	if err != nil {
		fmt.Printf("error copying tars into chroot dir: %v", err)
		os.Exit(1)
	}

	// Create /dev/null so that cmd.Run() doesn't complain
	err = utils.CreateDevNull(chrootDir)
	if err != nil {
		fmt.Printf("error creating /dev/null: %v", err)
		os.Exit(1)
	}

	err = syscall.Chroot(chrootDir)
	if err != nil {
		fmt.Printf("chroot err: %v", err)
		os.Exit(1)
	}

	cmd := exec.Command(command, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID,
	}

	err = cmd.Run()
	exitErr, ok := err.(*exec.ExitError)

	if ok {
		os.Exit(exitErr.ExitCode()) // The program exited with a non-zero exit code
	} else if err != nil {
		fmt.Printf("Err: %v", err)
		os.Exit(1)
	}
}

func cleanUp(chrootDir, tarDir string) {
	err := os.RemoveAll(tarDir)
	if err != nil {
		fmt.Printf("error removing tarDir: %v", err)
		os.Exit(1)
	}

	err = os.RemoveAll(chrootDir)
	if err != nil {
		fmt.Printf("error removing chrootDir: %v", err)
		os.Exit(1)
	}
}

func fetchDockerImageIntoDir(image, tarDir string) []string {
	repo, ref := utils.ParseImage(image)
	token, err := utils.AuthenticateWithDockerRegistry(repo)
	if err != nil {
		fmt.Printf("error pulling authenticating with docker registry: %v", err)
		os.Exit(1)
	}

	dockerClient := client.NewDockerClient(repo, ref, token)

	manifest, err := dockerClient.PullImageManifest()
	if err != nil {
		fmt.Printf("error pulling image manifest: %v", err)
		os.Exit(1)
	}

	tars, err := dockerClient.PullImageLayers(manifest, tarDir)
	if err != nil {
		fmt.Printf("error pulling image layers: %v", err)
		os.Exit(1)
	}

	return tars
}
