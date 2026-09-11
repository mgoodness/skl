package skl

// Fetcher abstracts fetching a GitHub source's contents onto local disk.
// Given a parsed GitHub source and a ref (empty meaning "the source's
// default branch"), it returns a local directory containing that ref's
// contents. The caller owns the returned directory and is responsible for
// removing it once done.
//
// Local-path sources bypass Fetcher entirely (see resolveSource): it
// exists solely so GitHub-based fetching can be faked in tests, keeping
// the bulk of the test suite network-free. GitHubFetcher is the
// production implementation; a small number of real-network tests
// exercise it directly, tagged "integration" and excluded from the
// default `go test` run (see github_fetcher_integration_test.go).
type Fetcher interface {
	Fetch(src GitHubSource, ref string) (dir string, err error)
}
