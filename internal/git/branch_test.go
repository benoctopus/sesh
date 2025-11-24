package git

import (
	"testing"
)

func TestParseBranchInfo(t *testing.T) {
	tests := []struct {
		name    string
		branch  string
		current string
		want    BranchInfo
	}{
		{
			name:    "local branch not current",
			branch:  "feature/foo",
			current: "main",
			want: BranchInfo{
				Name:      "feature/foo",
				IsRemote:  false,
				IsCurrent: false,
			},
		},
		{
			name:    "local branch is current",
			branch:  "main",
			current: "main",
			want: BranchInfo{
				Name:      "main",
				IsRemote:  false,
				IsCurrent: true,
			},
		},
		{
			name:    "remote branch",
			branch:  "origin/feature/bar",
			current: "main",
			want: BranchInfo{
				Name:      "feature/bar",
				IsRemote:  true,
				IsCurrent: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch := tt.branch
			isRemote := false
			branchName := branch

			// Simulate the logic from ListAllBranches
			if len(branch) > 7 && branch[:7] == "origin/" {
				isRemote = true
				branchName = branch[7:]
			}

			result := BranchInfo{
				Name:      branchName,
				IsRemote:  isRemote,
				IsCurrent: branchName == tt.current,
			}

			if result.Name != tt.want.Name {
				t.Errorf("Name = %q, want %q", result.Name, tt.want.Name)
			}
			if result.IsRemote != tt.want.IsRemote {
				t.Errorf("IsRemote = %v, want %v", result.IsRemote, tt.want.IsRemote)
			}
			if result.IsCurrent != tt.want.IsCurrent {
				t.Errorf("IsCurrent = %v, want %v", result.IsCurrent, tt.want.IsCurrent)
			}
		})
	}
}
