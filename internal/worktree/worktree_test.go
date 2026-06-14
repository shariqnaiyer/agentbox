package worktree

import "testing"

func TestPath(t *testing.T) {
	cases := []struct {
		repo, name, want string
	}{
		{"/Users/me/dev/myrepo", "task1", "/Users/me/dev/wt-task1"},
		{"/srv/code/app", "fix", "/srv/code/wt-fix"},
		{"/a/b/c/", "x", "/a/b/wt-x"}, // trailing slash: dirname drops the empty element
	}
	for _, c := range cases {
		if got := Path(c.repo, c.name); got != c.want {
			t.Errorf("Path(%q,%q) = %q, want %q", c.repo, c.name, got, c.want)
		}
	}
}
