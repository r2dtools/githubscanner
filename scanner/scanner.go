package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
)

const (
	GitHuhApi       = "https://api.github.com"
	perPage         = 100
	maxWorkersCount = 100
)

type ResultItem struct {
	Repository *Repository
	Releases   []*Release
}

type Repository struct {
	FullName string `json:"full_name"`
	Name     string `json:"name"`
}

type Release struct {
	Name string `json:"name"`
}

type Scanner struct {
	BaseUrl string
	PerPage int
}

func GetDefaultScanner() *Scanner {
	return &Scanner{
		BaseUrl: GitHuhApi,
		PerPage: perPage,
	}
}

func (s *Scanner) ScanRepositories(user string) (items []*ResultItem, err error) {
	repositories, err := s.GetAllRepositories(user)
	if err != nil {
		return
	}

	jobsCount := len(repositories)
	jobs := make(chan *Repository, jobsCount)
	results := make(chan *ResultItem, jobsCount)
	errors := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		close(results)
		close(errors)
	}()

	worker := func(jobs <-chan *Repository, results chan<- *ResultItem, errors chan<- error) {
		for repository := range jobs {
			select {
			case <-ctx.Done():
				return
			default:
			}
			releases, err := s.GetAllReleases(user, repository.Name)
			if err != nil {
				errors <- err
				cancel()
				return
			}
			item := &ResultItem{
				Repository: repository,
				Releases:   releases,
			}
			results <- item
		}
	}

	workersCount := maxWorkersCount
	if jobsCount < maxWorkersCount {
		workersCount = jobsCount
	}
	for i := 0; i < workersCount; i++ {
		go worker(jobs, results, errors)
	}

	for _, repository := range repositories {
		jobs <- repository
	}
	close(jobs)

	for i := 0; i < jobsCount; i++ {
		select {
		case err = <-errors:
			err = fmt.Errorf("could not scan repository for the account %s: %v", user, err)
			return
		case item := <-results:
			items = append(items, item)
		}
	}
	s.sortResultItems(items)

	return
}

func (s *Scanner) GetAllReleases(user, repository string) ([]*Release, error) {
	var releases []*Release
	page := 1
	for {
		releasesChunk, err := s.GetReleasesPerPage(user, repository, page)
		if err != nil {
			return nil, err
		}
		releases = append(releases, releasesChunk...)
		if len(releasesChunk) < s.getPerPage() {
			break
		}
		page++
	}

	return releases, nil
}

func (s *Scanner) GetReleasesPerPage(user, repository string, page int) ([]*Release, error) {
	if err := s.checkPage(page); err != nil {
		return nil, err
	}
	if err := s.checkUser(user); err != nil {
		return nil, err
	}
	if err := s.checkRepository(repository); err != nil {
		return nil, err
	}
	response, err := http.Get(fmt.Sprintf("%s/repos/%s/%s/releases?per_page=%d&page=%d", s.BaseUrl, user, repository, s.getPerPage(), page))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not get releases for the repository %s: %s", repository, s.getApiErrorMessage(response.Body, response.Status))
	}

	var releases []*Release
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

func (s *Scanner) GetAllRepositories(user string) ([]*Repository, error) {
	var repositories []*Repository
	page := 1
	for {
		repositoriesChunk, err := s.GetRepositoriesPerPage(user, page)
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, repositoriesChunk...)
		if len(repositoriesChunk) < s.getPerPage() {
			break
		}
		page++
	}

	return repositories, nil
}

func (s *Scanner) GetRepositoriesPerPage(user string, page int) ([]*Repository, error) {
	if err := s.checkPage(page); err != nil {
		return nil, err
	}
	if err := s.checkUser(user); err != nil {
		return nil, err
	}
	response, err := http.Get(fmt.Sprintf("%s/users/%s/repos?per_page=%d&page=%d", s.BaseUrl, user, s.getPerPage(), page))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("account %s does not exist", user)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not get repositories for the account %s: %s", user, s.getApiErrorMessage(response.Body, response.Status))
	}

	var repositories []*Repository
	if err := json.NewDecoder(response.Body).Decode(&repositories); err != nil {
		return nil, err
	}

	return repositories, nil
}

func (s *Scanner) sortResultItems(items []*ResultItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Repository.FullName < items[j].Repository.FullName
	})
}

func (s *Scanner) getPerPage() int {
	if s.PerPage <= 0 {
		return perPage
	}

	return s.PerPage
}

func (s *Scanner) checkPage(page int) error {
	if page < 1 {
		return errors.New("page could not be less than 1")
	}

	return nil
}

func (s *Scanner) checkUser(user string) error {
	if user == "" {
		return errors.New("user name could not be empty")
	}

	return nil
}

func (s *Scanner) checkRepository(repository string) error {
	if repository == "" {
		return errors.New("repository name could not be empty")
	}

	return nil
}

func (s *Scanner) getApiErrorMessage(reader io.Reader, defaultMessage string) string {
	mBytes, err := io.ReadAll(reader)
	if err != nil {
		return defaultMessage
	}

	message := struct {
		Message string `json:"message"`
	}{}
	if err := json.Unmarshal(mBytes, &message); err != nil {
		return defaultMessage
	}

	return message.Message
}
