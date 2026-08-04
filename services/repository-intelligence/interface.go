// Package repositoryintelligence declares the interface Step 3 locks so
// Step 5 can swap in a real implementation (querying .ai/ docs) with zero
// changes to any agent or activity that depends on it. Until Step 5, only a
// stub implementation returning canned data exists.
package repositoryintelligence

import (
	"context"

	baseactivity "github.com/khennicb/sw-factory/pkg/activity"
)

// DocRef, FileRef, and CodeRef are intentionally minimal placeholders —
// Step 5 is expected to flesh these out once the real .ai/ doc format and
// repository indexing strategy are decided.
type DocRef struct {
	Path    string
	Excerpt string
}

type FileRef struct {
	Path string
}

type CodeRef struct {
	Path        string
	Description string
}

// Context is everything an Implementation/Review/Test-Analysis agent shim
// needs to ground its work in this repository, gathered once during the
// COLLECT_CONTEXT state and threaded through subsequent activities.
type Context struct {
	Docs                   []DocRef
	Files                  []FileRef
	ArchitectureNotes      []string
	SimilarImplementations []CodeRef
	CodingConventions      string
}

// RepositoryIntelligence is the locked interface. See stub.go for the
// Step 1/3 canned-data implementation.
type RepositoryIntelligence interface {
	GetContext(ctx context.Context, taskID baseactivity.TaskID) (Context, error)
}
