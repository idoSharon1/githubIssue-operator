package config

import (
	_ "embed"
	"encoding/json"
)

type Config struct {
	AuthSecret struct {
		GithubSecretName    string `json:"githubSecretName"`
		GithubSecretKeyName string `json:"githubSecretKeyName"`
	}
	UserGithubToken string `json:"userGithubToken"`
	FinalizerKey    string `json:"finalizerKey"`
	EnvName         string `json:"envName"`
	RepoLabelKey    string `json:"repoLabelKey"`
	TitleLabelKey   string `json:"titleLabelKey"`
	GithubApi       struct {
		BaseUrl string `json:"baseUrl"`
	}
}

//go:embed config.json
var configFile []byte

func LoadConfig() (*Config, error) {
	var config Config
	if err := json.Unmarshal(configFile, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
