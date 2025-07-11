//go:build pro

package git

import (
	"claude-squad/log"
	"fmt"
	"time"
)

func NewTreeSource(repoPath string) WorktreeSource {
	return NewWorktreeCache(repoPath)
}

// WorktreeCache pre-makes "temp" worktrees in a background goroutine. When you call GetGitWorktree(branchName),
// it renames the temp worktree into the correct name and returns it. It also updates the path of the worktree
// to match the branch.
//
// Making new git worktrees is expensive, so the background goroutine saves a ton of time. Going from a few secs
// to a few ms.
type WorktreeCache struct {
	worktrees chan *GitWorktree
	repoPath  string
}

func NewWorktreeCache(repoPath string) *WorktreeCache {
	cache := &WorktreeCache{
		worktrees: make(chan *GitWorktree, 1),
		repoPath:  repoPath,
	}

	go cache.backgroundWorktreeCreator()

	return cache
}

// backgroundWorktreeCreator creates new worktrees in the background, sending them to
// the chan when ready.
func (c *WorktreeCache) backgroundWorktreeCreator() {
	every := log.NewEvery(5 * time.Second)
	for {
		tempSessionName := fmt.Sprintf("claudesquad-worktree-cached-temp-%d", time.Now().UnixNano())
		worktree, err := newGitWorktree(c.repoPath, tempSessionName)
		if err != nil {
			if every.ShouldLog() {
				log.ErrorLog.Printf("worktree cache failed to create worktree: %s", err)
			}
			continue
		}
		if err := worktree.Setup(); err != nil {
			if every.ShouldLog() {
				log.ErrorLog.Printf("worktree cache failed to setup worktree: %s", err)
			}
		}

		c.worktrees <- worktree
	}
}

func (c *WorktreeCache) GetGitWorktree(repoPath string, sessionName string) (*GitWorktree, error) {
	select {
	case worktree := <-c.worktrees:
		newBranchName, pathName := sessionNameToBranchAndPath(sessionName)

		// Move worktree path.
		if _, err := worktree.runGitCommand(
			worktree.repoPath, "worktree", "move", worktree.worktreePath, pathName); err != nil {
			log.ErrorLog.Printf("failed to move worktree %s from cache: %s", worktree.worktreePath, err)
			return newGitWorktree(repoPath, sessionName)
		}
		worktree.worktreePath = pathName

		// Rename the branch using git command
		if _, err := worktree.runGitCommand(
			worktree.repoPath, "branch", "-m", worktree.branchName, newBranchName); err != nil {
			log.ErrorLog.Printf("failed to get tree %s from cache: %s\n", worktree.branchName, err)
			return newGitWorktree(repoPath, sessionName)
		}
		worktree.branchName = newBranchName

		return worktree, nil
	default:
		return newGitWorktree(repoPath, sessionName)
	}
}
