package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "git-split-diffs")
	if err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		fmt.Printf("Cleaning up temporary directory: %s\n", tempDir)
		os.RemoveAll(tempDir)
	}()
	fmt.Printf("Created temporary directory: %s\n", tempDir)

	// Clone the repository with minimum depth
	fmt.Println("Cloning git-split-diffs repository with minimum depth...")
	cmd := exec.Command("git", "clone", "--depth=1", "https://github.com/kujtimiihoxha/git-split-diffs", tempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error cloning repository: %v\n", err)
		os.Exit(1)
	}

	// Run npm install
	fmt.Println("Running npm install...")
	cmdNpmInstall := exec.Command("npm", "install")
	cmdNpmInstall.Dir = tempDir
	cmdNpmInstall.Stdout = os.Stdout
	cmdNpmInstall.Stderr = os.Stderr
	if err := cmdNpmInstall.Run(); err != nil {
		fmt.Printf("Error running npm install: %v\n", err)
		os.Exit(1)
	}

	// Run npm run build
	fmt.Println("Running npm run build...")
	cmdNpmBuild := exec.Command("npm", "run", "build")
	cmdNpmBuild.Dir = tempDir
	cmdNpmBuild.Stdout = os.Stdout
	cmdNpmBuild.Stderr = os.Stderr
	if err := cmdNpmBuild.Run(); err != nil {
		fmt.Printf("Error running npm run build: %v\n", err)
		os.Exit(1)
	}

	destDir := filepath.Join(".", "internal", "assets", "diff")
	destFile := filepath.Join(destDir, "index.mjs")

	// Make sure the destination directory exists
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		fmt.Printf("Error creating destination directory: %v\n", err)
		os.Exit(1)
	}

	// Copy the file
	srcFile := filepath.Join(tempDir, "build", "index.mjs")
	fmt.Printf("Copying %s to %s\n", srcFile, destFile)
	if err := copyFile(srcFile, destFile); err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully completed the process!")
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Make sure the file is written to disk
	err = destFile.Sync()
	if err != nil {
		return err
	}

	return nil
}
