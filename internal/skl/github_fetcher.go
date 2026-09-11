package skl

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GitHubFetcher is the production Fetcher implementation. It downloads src
// at ref (or src's default branch, via codeload's "HEAD" pseudo-ref, when
// ref is empty) as a tarball from codeload.github.com — the same
// unauthenticated archive endpoint GitHub's own "Download ZIP" button and
// tools like degit use — and extracts it into a fresh temp directory,
// stripping the tarball's single top-level "<repo>-<ref>/" wrapper
// directory so the returned directory is the repository root itself.
type GitHubFetcher struct {
	// HTTPClient is used for the archive download. A nil value (the
	// zero GitHubFetcher) uses a client with a 60-second timeout.
	HTTPClient *http.Client
}

const githubFetchTimeout = 60 * time.Second

func (f *GitHubFetcher) httpClient() *http.Client {
	if f.HTTPClient != nil {
		return f.HTTPClient
	}
	return &http.Client{Timeout: githubFetchTimeout}
}

// Fetch implements Fetcher.
func (f *GitHubFetcher) Fetch(src GitHubSource, ref string) (string, error) {
	archiveRef := ref
	if archiveRef == "" {
		archiveRef = "HEAD"
	}

	url := fmt.Sprintf("https://codeload.github.com/%s/%s/tar.gz/%s", src.Owner, src.Repo, archiveRef)
	resp, err := f.httpClient().Get(url)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", src, err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		if ref != "" {
			return "", fmt.Errorf("%s not found at ref %q: it may not exist, the ref may not exist, or it may be a private repository you don't have access to", src, ref)
		}
		return "", fmt.Errorf("%s not found: it may not exist, or it may be a private repository you don't have access to", src)
	default:
		return "", fmt.Errorf("fetching %s: unexpected response %s", src, resp.Status)
	}

	dir, err := os.MkdirTemp("", "skl-github-*")
	if err != nil {
		return "", fmt.Errorf("creating temp directory for %s: %w", src, err)
	}

	if err := extractTarGz(resp.Body, dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("extracting %s: %w", src, err)
	}

	return dir, nil
}

// extractTarGz extracts a gzip-compressed tar stream into dest, stripping
// each entry's first path segment: GitHub's codeload tarballs wrap all
// content in a single top-level "<repo>-<ref>/" directory, and stripping
// it here makes dest the repository root, matching a local source's
// shape (a directory whose immediate contents are the repo's own files).
func extractTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("reading gzip stream: %w", err)
	}
	defer func() { _ = gz.Close() }()

	cleanDest := filepath.Clean(dest)

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		name := hdr.Name
		if i := strings.IndexByte(name, '/'); i >= 0 {
			name = name[i+1:]
		} else {
			name = ""
		}
		if name == "" {
			// The top-level wrapper directory entry itself.
			continue
		}

		target := filepath.Join(dest, filepath.FromSlash(name))
		if target != cleanDest && !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
			return fmt.Errorf("tar entry %q escapes destination directory", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("creating %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("creating %s: %w", filepath.Dir(target), err)
			}
			if err := writeTarFile(target, tr, hdr.FileInfo().Mode()); err != nil {
				return err
			}
		default:
			// Symlinks and other special entry types are skipped: a skill's
			// SKILL.md and resource files are always regular files/directories.
		}
	}
}

func writeTarFile(target string, r io.Reader, mode os.FileMode) error {
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm()|0o200)
	if err != nil {
		return fmt.Errorf("creating %s: %w", target, err)
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, r); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}
	return nil
}
