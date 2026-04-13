package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/k1LoW/mo/internal/server"
	"github.com/mattn/go-isatty"
)

// isStdinPipe reports whether stdin is a pipe (not a terminal).
func isStdinPipe() bool {
	fd := os.Stdin.Fd()
	return !isatty.IsTerminal(fd) && !isatty.IsCygwinTerminal(fd)
}

// readStdin reads all content from the given reader and returns
// a generated name in the format "stdin-<hash>.md" along with the content.
func readStdin(r io.Reader) (name string, content string, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", "", fmt.Errorf("failed to read stdin: %w", err)
	}
	c := string(data)
	return stdinName(c), c, nil
}

// stdinName generates a deterministic name for stdin content
// in the format "stdin-<first 7 hex chars of SHA-256>.md".
func stdinName(content string) string {
	h := sha256.Sum256([]byte(content))
	return "stdin-" + hex.EncodeToString(h[:])[:7] + ".md"
}

// postUploadedFile uploads in-memory content to a running mo server.
func postUploadedFile(client *http.Client, addr, group, name, content string) (deeplinkEntry, error) {
	body, err := json.Marshal(server.UploadedFileData{
		Name:    name,
		Content: content,
	})
	if err != nil {
		return deeplinkEntry{}, err
	}
	resp, err := client.Post(
		fmt.Sprintf("http://%s/_/api/groups/%s/files/upload", addr, url.PathEscape(group)),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return deeplinkEntry{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return deeplinkEntry{}, fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}
	var entry server.FileEntry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return deeplinkEntry{}, err
	}
	return deeplinkEntry{
		URL:  buildDeeplink(addr, group, entry.ID),
		Name: entry.Name,
	}, nil
}
