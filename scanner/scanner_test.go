package scanner

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetAllRepositoriesSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/test/repos" {
			page := r.URL.Query().Get("page")
			w.WriteHeader(http.StatusOK)

			if page == "1" {
				w.Write([]byte(`[
				{"full_name": "test/repo1", "name": "repo1"},
				{"full_name": "test/repo2", "name": "repo2"},
				{"full_name": "test/repo3", "name": "repo3"}
				]`))
			} else {
				w.Write([]byte(`[
				{"full_name": "test/repo4", "name": "repo4"},
				{"full_name": "test/repo5", "name": "repo5"}
				]`))
			}
		}
	}))
	defer server.Close()

	scanner := Scanner{
		BaseUrl: server.URL,
		PerPage: 3,
	}
	repositories, err := scanner.GetAllRepositories("test")
	if err != nil {
		t.Fatal(err)
	}

	repoNames := []string{}
	for _, reporepository := range repositories {
		repoNames = append(repoNames, reporepository.FullName)
	}
	expectedRepoNames := []string{"test/repo1", "test/repo2", "test/repo3", "test/repo4", "test/repo5"}

	if !equal(repoNames, expectedRepoNames) {
		t.Fatalf("invalid repositories list, expected %v, got %v", expectedRepoNames, repoNames)
	}
}

func TestGetAllRepositoriesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/test/repos" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message": "forbidden"}`))
		}
	}))
	defer server.Close()

	scanner := Scanner{
		BaseUrl: server.URL,
	}
	_, err := scanner.GetAllRepositories("test")
	if err == nil {
		t.Fatal("invalid response for repository list: error is expected")
	}

	if !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("invalid error message, expected 'forbidden', got %s", err.Error())
	}
}

func TestGetAllReleasesSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test/test/releases" {
			page := r.URL.Query().Get("page")
			w.WriteHeader(http.StatusOK)

			if page == "1" {
				w.Write([]byte(`[
				{"name": "repo5"},
				{"name": "repo4"},
				{"name": "repo3"}
				]`))
			} else {
				w.Write([]byte(`[
				{"name": "repo2"},
				{"name": "repo1"}
				]`))
			}
		}
	}))
	defer server.Close()

	scanner := Scanner{
		BaseUrl: server.URL,
		PerPage: 3,
	}
	releases, err := scanner.GetAllReleases("test", "test")
	if err != nil {
		t.Fatal(err)
	}

	names := []string{}
	for _, release := range releases {
		names = append(names, release.Name)
	}
	expectedNames := []string{"repo5", "repo4", "repo3", "repo2", "repo1"}

	if !equal(names, expectedNames) {
		t.Fatalf("invalid releases list for a repository, expected %v, got %v", expectedNames, names)
	}
}

func TestGetAllReleasesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test/test/releases" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message": "forbidden"}`))
		}
	}))
	defer server.Close()

	scanner := Scanner{
		BaseUrl: server.URL,
	}
	_, err := scanner.GetAllReleases("test", "test")
	if err == nil {
		t.Fatal("invalid response for release list: error is expected")
	}

	if !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("invalid error message, expected 'forbidden', got %s", err.Error())
	}
}

func TestScanRepositoriesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/test/repos" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{"full_name": "test/test", "name": "test"}
				]`))
		}
		if r.URL.Path == "/repos/test/test/releases" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message": "forbidden"}`))
		}
	}))
	defer server.Close()

	scanner := Scanner{
		BaseUrl: server.URL,
		PerPage: 3,
	}
	_, err := scanner.ScanRepositories("test")
	if err == nil {
		t.Fatal("invalid response for during scanning of repositories: error is expected")
	}

	if !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("invalid error message, expected 'forbidden', got %s", err.Error())
	}
}

func TestScanRepositoriesSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/test/repos" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{"full_name": "test/test", "name": "test"}
				]`))
		}
		if r.URL.Path == "/repos/test/test/releases" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{"name": "test"}
				]`))
		}
	}))
	defer server.Close()

	scanner := Scanner{
		BaseUrl: server.URL,
		PerPage: 100,
	}
	items, err := scanner.ScanRepositories("test")
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("invalid scanned repositories items count, expected 1, got %d", len(items))
	}

	item := items[0]
	if item.Repository.FullName != "test/test" {
		t.Fatalf("invalid repository full name, expected 'test/test', got %s", item.Repository.FullName)
	}
	if item.Repository.Name != "test" {
		t.Fatalf("invalid repository name, expected 'test', got %s", item.Repository.Name)
	}
	if len(item.Releases) != 1 {
		t.Fatalf("invalid releases count for the scanned repository, expected 1, got %d", len(item.Releases))
	}
	if item.Releases[0].Name != "test" {
		t.Fatalf("invalid release name for the scanned repository, expected 'test', got %s", item.Releases[0].Name)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
