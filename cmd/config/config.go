package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Config struct {
	AuthSecret struct {
		GithubSecretName    string `json:"githubSecretName"`
		GithubSecretKeyName string `json:"githubSecretKeyName"`
	}
	UserGithubToken string `json:"userGithubToken"`
	FinalizerKey    string `json:"finalizerKey"`
	EnvName         string `json:"envName"`
	GithubApi       struct {
		BaseUrl string `json:"baseUrl"`
	}
}

func LoadConfig() (*Config, error) {
	file, err := os.Open("/mnt/c/ido-proj/githubIssue-operator/cmd/config/config.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
