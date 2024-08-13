package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
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

func LoadEnvFile(envFile string) error {
	err := godotenv.Load(dir(envFile))
	if err != nil {
		return err
	}

	return nil
}

func dir(envFile string) string {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	print(currentDir)

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			break
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			panic(fmt.Errorf("go.mod not found"))
		}
		currentDir = parent
	}

	return filepath.Join(currentDir, envFile)
}
