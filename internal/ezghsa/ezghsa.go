package ezghsa

import (
	"context"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/google/go-github/v51/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	OAuthToken string `yaml:"oauth_token"`
}

func Hosts() (map[string]Config, error) {
	home, _ := os.LookupEnv("HOME")
	p := path.Join(home, ".config", "ezghsa", "hosts.yml")

	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)

	m := map[string]Config{}
	err = decoder.Decode(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func getToken() string {
	token, _ := os.LookupEnv("GITHUB_TOKEN")
	if token != "" {
		return token
	}

	h, err := Hosts()
	if err != nil {
		log.Fatalf("invalid host configuration: %v", err)
	}

	return h["github.com"].OAuthToken
}

func DefaultHttpClient() *http.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: getToken()},
	)
	return oauth2.NewClient(ctx, ts)
}

type App struct {
	client *github.Client
	user   *github.User
}

func New(httpClient *http.Client) *App {
	client := github.NewClient(httpClient)

	if client.UserAgent != "" {
		client.UserAgent += " "
	}
	client.UserAgent += "ezghsa (+https://github.com/csdev/ezghsa)"

	return &App{
		client: client,
	}
}

func (app *App) loadUser() error {
	if app.user != nil {
		return nil
	}

	user, _, err := app.client.Users.Get(context.Background(), "")
	if err != nil {
		return err
	}

	app.user = user
	return nil
}

func (app *App) GetMyRepos() ([]*github.Repository, error) {
	opts := &github.RepositoryListOptions{
		Affiliation: "owner",
	}
	repos, _, err := app.client.Repositories.List(context.Background(), "", opts)
	return repos, err
}

func (app *App) GetRepos(ownerRepoNames []string) ([]*github.Repository, error) {
	repos := make([]*github.Repository, 0, len(ownerRepoNames))
	for _, ownerRepoName := range ownerRepoNames {
		ownerName, repoName, ok := strings.Cut(ownerRepoName, "/")
		if !ok {
			// if "owner/" is omitted, default to the name of the current user
			// (matches the behavior of gh)
			err := app.loadUser()
			if err != nil {
				return nil, err
			}

			ownerName = app.user.GetLogin()
			repoName = ownerRepoName
		}

		repo, _, err := app.client.Repositories.Get(context.Background(), ownerName, repoName)
		if err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}
	return repos, nil
}

func (app *App) CheckAlertsEnabled(ownerName string, repoName string) (bool, error) {
	isEnabled, _, err := app.client.Repositories.GetVulnerabilityAlerts(
		context.Background(), ownerName, repoName)
	return isEnabled, err
}

func (app *App) GetOpenAlerts(ownerName string, repoName string) ([]*github.DependabotAlert, error) {
	opts := &github.ListAlertsOptions{
		State: github.String("open"),
	}
	alerts, _, err := app.client.Dependabot.ListRepoAlerts(context.Background(), ownerName, repoName, opts)
	return alerts, err
}

func FilterAlerts(alerts []*github.DependabotAlert, fn func(*github.DependabotAlert) bool) []*github.DependabotAlert {
	selectedAlerts := make([]*github.DependabotAlert, 0, len(alerts))

	for _, alert := range alerts {
		if fn(alert) {
			selectedAlerts = append(selectedAlerts, alert)
		}
	}
	return selectedAlerts
}
