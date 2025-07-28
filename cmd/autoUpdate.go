/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// autoUpdateCmd represents the autoUpdate command
var autoUpdateCmd = &cobra.Command{
	Use:   "auto-update",
	Short: "Set up a cron job to auto-update default branch with upstream",
	Run: func(cmd *cobra.Command, args []string) {
		promptAndStartCron()
	},
}

func promptAndStartCron() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter path to the project: ")
	path, _ := reader.ReadString('\n')

	branches, err := getRemoteBranches(path)
	if err != nil {
		fmt.Print("Error getting branches")
		os.Exit(1)
	}
	branch, _ := pterm.DefaultInteractiveSelect.WithOptions(branches).WithDefaultText("Select default branch").Show()

	remotes, err := getRemotes(path)
	if err != nil {
		fmt.Print("Error getting remotes")
		os.Exit(1)
	}
	upstream, _ := pterm.DefaultInteractiveSelect.WithOptions(remotes).WithDefaultText("Select Remote to mirror").Show()

	fmt.Print("Run every how many hours? (e.g., 1): ")
	interval, _ := reader.ReadString('\n')

	// Sanitize inputs
	branch = strings.TrimSpace(branch)
	upstream = strings.TrimSpace(upstream)
	path = strings.TrimSpace(path)
	interval = strings.TrimSpace(interval)

	scriptContent := fmt.Sprintf(`#!/bin/bash
set -e  # Exit on any error

REPO_PATH="%s"

UPSTREAM_URL=$(git -C "$REPO_PATH" remote get-url %s)

check_git_connectivity() {
    if ! git -C "$REPO_PATH" ls-remote "$UPSTREAM_URL" >/dev/null 2>&1; then
        echo "Unable to reach remote $UPSTREAM_URL. Skipping sync at $(date)"
        exit 0
    fi
}

# Function to check if we can safely switch branches
can_switch_branch() {
    # Check if there are uncommitted changes
    if ! git -C "$REPO_PATH" diff-index --quiet HEAD --; then
        echo "Uncommitted changes detected. Skipping sync to avoid data loss at $(date)"
        exit 0
    fi
    
    # Check if there are untracked files that would be overwritten
    if [ -n "$(git -C "$REPO_PATH" ls-files --others --exclude-standard)" ]; then
        echo "Untracked files detected. Please review before sync at $(date)"
        exit 0
    fi
}

# Check internet connectivity first
check_git_connectivity

# Store current branch
CURRENT=$(git -C "$REPO_PATH" rev-parse --abbrev-ref HEAD)
echo "Current branch: $CURRENT"

# If we're already on the target branch, just update it
if [ "$CURRENT" = "%s" ]; then
    echo "Already on target branch %s"
    git -C "$REPO_PATH" fetch %s || {
        echo "Failed to fetch from remote. Check connectivity at $(date)"
        exit 1
    }
    
    # Check if we're behind the remote
    LOCAL=$(git -C "$REPO_PATH" rev-parse HEAD)
    REMOTE=$(git -C "$REPO_PATH" rev-parse %s/%s)
    
    if [ "$LOCAL" = "$REMOTE" ]; then
        echo "Already up to date at $(date)"
        exit 0
    fi
    
    # Try to rebase, but handle conflicts gracefully
    if ! git -C "$REPO_PATH" rebase %s/%s; then
        echo "Rebase failed due to conflicts. Manual intervention required at $(date)"
        git -C "$REPO_PATH" rebase --abort
        exit 1
    fi
else
    # We're on a different branch - check if we can safely switch
    can_switch_branch
    
    # Perform git operations
    echo "Switching from $CURRENT to %s"
    git -C "$REPO_PATH" checkout %s || {
        echo "Failed to checkout %s at $(date)"
        exit 1
    }
    
    git -C "$REPO_PATH" fetch %s || {
        echo "Failed to fetch from remote. Check connectivity at $(date)"
        git -C "$REPO_PATH" checkout $CURRENT  # Return to original branch
        exit 1
    }
    
    # Try to rebase, but handle conflicts gracefully
    if ! git -C "$REPO_PATH" rebase %s/%s; then
        echo "Rebase failed due to conflicts. Manual intervention required at $(date)"
        git -C "$REPO_PATH" rebase --abort
        git -C "$REPO_PATH" checkout $CURRENT  # Return to original branch
        exit 1
    fi
    
    # Return to original branch
    git -C "$REPO_PATH" checkout $CURRENT || {
        echo "Warning: Failed to return to original branch $CURRENT at $(date)"
        exit 1
    }
    echo "Returned to original branch: $CURRENT"
fi

echo "Git sync completed successfully at $(date)"
`, path, upstream, branch, branch, upstream, upstream, branch, upstream, branch, branch, branch, branch, upstream, upstream, branch)

	// Create script file path
	scriptPath := "/tmp/git-sync-" + strings.ReplaceAll(filepath.Base(path), " ", "-") + ".sh"
	
	// Write script to file
	err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	if err != nil {
		fmt.Printf("Error creating script file: %v\n", err)
		return
	}

	// Create cron job that calls the script
	job := fmt.Sprintf(`0 */%s * * * %s >> /tmp/git-sync.log 2>&1`, interval, scriptPath)

	cmd := exec.Command("bash", "-c", fmt.Sprintf(`(crontab -l 2>/dev/null; echo "%s") | crontab -`, job))
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error adding cron job: %v\n", err)
		// Clean up script file if cron job creation failed
		os.Remove(scriptPath)
	} else {
		fmt.Printf("✅ Cron job added successfully.\n")
		fmt.Printf("📝 Script created at: %s\n", scriptPath)
		fmt.Printf("📋 Logs will be written to: /tmp/git-sync.log\n")
		fmt.Printf("🌿 Selected branch: %s\n", branch)
		fmt.Printf("🔗 Selected remote: %s\n", upstream)
		fmt.Printf("⏰ Will run every %s hours\n", interval)
		fmt.Printf("\n⚠️  Important notes:\n")
		fmt.Printf("   • Sync will be skipped if you have uncommitted changes\n")
		fmt.Printf("   • Sync will be skipped if there's no internet connection\n")
		fmt.Printf("   • If conflicts occur, manual intervention will be required\n")
		fmt.Printf("   • Check logs at /tmp/git-sync.log for details\n")
	}
}

func getRemotes(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	cmd := exec.Command("git", "-C", path,  "remote")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var remotes []string
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			remotes = append(remotes, line)
		}
	}
	
	if len(remotes) == 0 {
		return nil, fmt.Errorf("no remotes found")
	}
	
	return remotes, nil
}

func getRemoteBranches(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	cmd := exec.Command("git", "-C", path, "branch", "-a")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error getting remote branches", err)
		return nil, err
	}

	var branches []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Remove the * indicator for current branch
		if strings.HasPrefix(line, "* ") {
			line = strings.TrimPrefix(line, "* ")
		}
		
		// Skip remote tracking branches in the main list (they start with remotes/)
		if strings.HasPrefix(line, "remotes/") {
			continue
		}
		
		// Skip HEAD references
		if strings.Contains(line, "HEAD ->") {
			continue
		}
		
		if line != "" {
			branches = append(branches, line)
		}
	}
	
	return branches, nil


	
}

func init() {
	rootCmd.AddCommand(autoUpdateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// autoUpdateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// autoUpdateCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
