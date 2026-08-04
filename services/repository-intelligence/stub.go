package repositoryintelligence

import (
	"context"

	baseactivity "github.com/khennicb/sw-factory/pkg/activity"
)

// Stub is a canned-data implementation of RepositoryIntelligence. It exists
// so the Task Workflow's COLLECT_CONTEXT state has something to call
// starting in Step 3, without depending on the real .ai/ indexing work
// planned for Step 5. Swapping Stub for the real implementation requires no
// changes to any caller — that's the point of locking the interface here.
type Stub struct{}

func NewStub() *Stub { return &Stub{} }

func (s *Stub) GetContext(ctx context.Context, taskID baseactivity.TaskID) (Context, error) {
	return Context{
		Docs:                   []DocRef{{Path: ".ai/docs/README.md", Excerpt: "canned stub excerpt"}},
		Files:                  []FileRef{{Path: "README.md"}},
		ArchitectureNotes:      []string{"stub: no real architecture analysis yet (Step 5)"},
		SimilarImplementations: nil,
		CodingConventions:      "stub: conventions not yet indexed (Step 5)",
	}, nil
}
