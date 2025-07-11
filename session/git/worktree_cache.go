//go:build !pro

package git

func NewTreeSource(_ string) WorktreeSource {
	return &SimpleWorktreeSource{}
}

// SimpleWorktreeSource is a passthrough for newGitWorktree.
type SimpleWorktreeSource struct{}

func (s *SimpleWorktreeSource) GetGitWorktree(repoPath string, sessionName string) (
	*GitWorktree,
	error,
) {
	worktree, err := newGitWorktree(repoPath, sessionName)
	if err != nil {
		return nil, err
	}
	return worktree.Setup()
}
