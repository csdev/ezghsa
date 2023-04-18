package ezghsa

import (
	"context"
	"log"
	"os"
	"path"

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

func Client() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: getToken()},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	if client.UserAgent != "" {
		client.UserAgent += " "
	}
	client.UserAgent += "ezghsa (+https://github.com/csdev/ezghsa)"

	return client
}
