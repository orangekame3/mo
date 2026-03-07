package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k1LoW/mo/internal/server"
)

func TestResolvePatterns_NoGlobChars(t *testing.T) {
	_, err := resolvePatterns([]string{"README.md"})
	if err == nil {
		t.Fatal("resolvePatterns should return error for pattern without glob chars")
	}
}

func TestResolvePatterns_Valid(t *testing.T) {
	patterns, err := resolvePatterns([]string{"**/*.md", "docs/*.md"})
	if err != nil {
		t.Fatalf("resolvePatterns returned error: %v", err)
	}
	if len(patterns) != 2 {
		t.Fatalf("got %d patterns, want 2", len(patterns))
	}
	for _, p := range patterns {
		if !filepath.IsAbs(p) {
			t.Errorf("pattern %q is not absolute", p)
		}
	}
}

func TestRun_UnwatchWithWatch(t *testing.T) {
	unwatchPatterns = []string{"**/*.md"}
	watchPatterns = []string{"**/*.md"}
	defer func() {
		unwatchPatterns = nil
		watchPatterns = nil
	}()

	err := run(rootCmd, nil)
	if err == nil {
		t.Fatal("run should return error when --unwatch and --watch are both specified")
	}
	want := "cannot use --unwatch with --watch"
	if err.Error() != want {
		t.Fatalf("got error %q, want %q", err.Error(), want)
	}
}

func TestRun_UnwatchWithArgs(t *testing.T) {
	unwatchPatterns = []string{"**/*.md"}
	defer func() { unwatchPatterns = nil }()

	err := run(rootCmd, []string{"README.md"})
	if err == nil {
		t.Fatal("run should return error when --unwatch and args are both specified")
	}
	want := "cannot use --unwatch with file arguments"
	if err.Error() != want {
		t.Fatalf("got error %q, want %q", err.Error(), want)
	}
}

func TestRun_WatchWithArgs(t *testing.T) {
	t.Run("with glob pattern", func(t *testing.T) {
		watchPatterns = []string{"**/*.md"}
		defer func() { watchPatterns = nil }()

		err := run(rootCmd, []string{"README.md"})
		if err == nil {
			t.Fatal("run should return error when --watch and args are both specified")
		}
		want := "cannot use --watch (-w) with file arguments"
		if err.Error() != want {
			t.Fatalf("got error %q, want %q", err.Error(), want)
		}
	})

	t.Run("without glob chars hints shell expansion", func(t *testing.T) {
		watchPatterns = []string{"README.md"}
		defer func() { watchPatterns = nil }()

		err := run(rootCmd, []string{"CHANGELOG.md"})
		if err == nil {
			t.Fatal("run should return error when --watch and args are both specified")
		}
		if !strings.Contains(err.Error(), "shell may have expanded") {
			t.Fatalf("error should hint shell expansion, got %q", err.Error())
		}
	})
}

func TestMergeGroups(t *testing.T) {
	t.Run("restored files come first, CLI files appended after", func(t *testing.T) {
		base := map[string][]string{"default": {"/a.md", "/b.md"}}
		additional := map[string][]string{"default": {"/c.md"}}
		got := mergeGroups(base, additional)
		want := []string{"/a.md", "/b.md", "/c.md"}
		if len(got["default"]) != len(want) {
			t.Fatalf("got %v, want %v", got["default"], want)
		}
		for i, v := range want {
			if got["default"][i] != v {
				t.Fatalf("got[%d] = %s, want %s", i, got["default"][i], v)
			}
		}
	})

	t.Run("deduplicates files in the same group", func(t *testing.T) {
		base := map[string][]string{"default": {"/a.md", "/b.md"}}
		additional := map[string][]string{"default": {"/b.md", "/c.md"}}
		got := mergeGroups(base, additional)
		want := []string{"/a.md", "/b.md", "/c.md"}
		if len(got["default"]) != len(want) {
			t.Fatalf("got %v, want %v", got["default"], want)
		}
		for i, v := range want {
			if got["default"][i] != v {
				t.Fatalf("got[%d] = %s, want %s", i, got["default"][i], v)
			}
		}
	})

	t.Run("merges different groups", func(t *testing.T) {
		base := map[string][]string{"docs": {"/doc.md"}}
		additional := map[string][]string{"default": {"/a.md"}}
		got := mergeGroups(base, additional)
		if len(got["docs"]) != 1 || got["docs"][0] != "/doc.md" {
			t.Fatalf("got docs=%v, want [/doc.md]", got["docs"])
		}
		if len(got["default"]) != 1 || got["default"][0] != "/a.md" {
			t.Fatalf("got default=%v, want [/a.md]", got["default"])
		}
	})

	t.Run("nil base returns additional only", func(t *testing.T) {
		additional := map[string][]string{"default": {"/a.md"}}
		got := mergeGroups(nil, additional)
		if len(got["default"]) != 1 || got["default"][0] != "/a.md" {
			t.Fatalf("got %v, want [/a.md]", got["default"])
		}
	})

	t.Run("nil additional returns base only", func(t *testing.T) {
		base := map[string][]string{"default": {"/a.md"}}
		got := mergeGroups(base, nil)
		if len(got["default"]) != 1 || got["default"][0] != "/a.md" {
			t.Fatalf("got %v, want [/a.md]", got["default"])
		}
	})

	t.Run("both nil returns nil", func(t *testing.T) {
		got := mergeGroups(nil, nil)
		if got != nil {
			t.Fatalf("got %v, want nil", got)
		}
	})
}

func TestFilterValidRestoreData(t *testing.T) {
	t.Run("keeps only existing files", func(t *testing.T) {
		dir := t.TempDir()
		existing := filepath.Join(dir, "a.md")
		os.WriteFile(existing, []byte("# A"), 0o600) //nolint:errcheck
		missing := filepath.Join(dir, "missing.md")

		rd := &server.RestoreData{
			Groups: map[string][]string{
				"default": {existing, missing},
			},
		}

		filesByGroup, _ := filterValidRestoreData(rd)
		if len(filesByGroup["default"]) != 1 {
			t.Fatalf("got %d files, want 1", len(filesByGroup["default"]))
		}
		if filesByGroup["default"][0] != existing {
			t.Fatalf("got %s, want %s", filesByGroup["default"][0], existing)
		}
	})

	t.Run("omits group when all files missing", func(t *testing.T) {
		rd := &server.RestoreData{
			Groups: map[string][]string{
				"docs": {"/nonexistent/a.md", "/nonexistent/b.md"},
			},
		}

		filesByGroup, _ := filterValidRestoreData(rd)
		if _, ok := filesByGroup["docs"]; ok {
			t.Fatal("group with all missing files should not appear in result")
		}
	})

	t.Run("passes patterns through unchanged", func(t *testing.T) {
		rd := &server.RestoreData{
			Groups: map[string][]string{},
			Patterns: map[string][]string{
				"default": {"/some/path/*.md"},
			},
		}

		_, patternsByGroup := filterValidRestoreData(rd)
		if len(patternsByGroup["default"]) != 1 {
			t.Fatalf("got %d patterns, want 1", len(patternsByGroup["default"]))
		}
		if patternsByGroup["default"][0] != "/some/path/*.md" {
			t.Fatalf("got %s, want /some/path/*.md", patternsByGroup["default"][0])
		}
	})

	t.Run("empty restore data returns empty results", func(t *testing.T) {
		rd := &server.RestoreData{}

		filesByGroup, patternsByGroup := filterValidRestoreData(rd)
		if len(filesByGroup) != 0 {
			t.Fatalf("got %d groups, want 0", len(filesByGroup))
		}
		if len(patternsByGroup) != 0 {
			t.Fatalf("got %d pattern groups, want 0", len(patternsByGroup))
		}
	})
}
