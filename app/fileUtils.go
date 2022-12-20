package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

func copyExecutableIntoDir(chrootDir string, executablePath string) error {
	executablePathInChrootDir := path.Join(chrootDir, executablePath)

	err := os.MkdirAll(path.Dir(executablePathInChrootDir), 0750)
	if err != nil {
		return err
	}

	return copyFile(executablePath, executablePathInChrootDir)
}

func copyFile(sourceFilePath, destinationFilePath string) error {
	sourceFileStat, err := os.Stat(sourceFilePath)
	if err != nil {
		return err
	}

	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.OpenFile(destinationFilePath, os.O_RDWR|os.O_CREATE, sourceFileStat.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func createDevNull(chrootDir string) error {
	if err := os.MkdirAll(path.Join(chrootDir, "dev"), 0750); err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(chrootDir, "dev", "null"), []byte{}, 0644)
}

func extractTarsToDir(chootDir string, paths []string) error {
	for _, path := range paths {
		cmd := exec.Command("tar", "xf", path, "-C", chootDir)
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}
