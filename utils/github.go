package utils

type GitHubRecipientError struct {
	username string
}

func (g GitHubRecipientError) Username() string {
	return g.username
}

func (GitHubRecipientError) Error() string {
	return `"github:" recipients were removed from the design`
}
