package main

import (
	"encoding/base64"
	"log"
	"os"
	"path"
	"strings"

	docker "github.com/drone-plugins/drone-buildx"
	"github.com/drone-plugins/drone-buildx-gcr/internal/gcp"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Repo        string
	Registry    string
	Password    string
	Username    string
	AccessToken string
}

func loadConfig() Config {
	// Default username
	username := "_json_key"
	var config Config

	// Load env-file if it exists
	if env := os.Getenv("PLUGIN_ENV_FILE"); env != "" {
		if err := godotenv.Load(env); err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}

	idToken := getenv("PLUGIN_OIDC_TOKEN_ID")
	projectId := getenv("PLUGIN_PROJECT_NUMBER")
	poolId := getenv("PLUGIN_POOL_ID")
	providerId := getenv("PLUGIN_PROVIDER_ID")
	serviceAccountEmail := getenv("PLUGIN_SERVICE_ACCOUNT_EMAIL")

	if idToken != "" && projectId != "" && poolId != "" && providerId != "" && serviceAccountEmail != "" {
		federalToken, err := gcp.GetFederalToken(idToken, projectId, poolId, providerId)
		if err != nil {
			logrus.Fatalf("Error getting federal token: %s", err)
		}
		accessToken, err := gcp.GetGoogleCloudAccessToken(federalToken, serviceAccountEmail)
		if err != nil {
			logrus.Fatalf("Error getting Google Cloud Access Token: %s", err)
		}
		config.AccessToken = accessToken
	} else {
		password := getenv(
			"PLUGIN_JSON_KEY",
			"GCR_JSON_KEY",
			"GOOGLE_CREDENTIALS",
			"TOKEN",
		)
		// decode the token if base64 encoded
		decoded, err := base64.StdEncoding.DecodeString(password)
		if err == nil {
			password = string(decoded)
		}
		config.Password = password
	}
	config.Username = username
	config.Repo = getenv("PLUGIN_REPO")
	config.Registry = getenv("PLUGIN_REGISTRY")

	return config
}

func main() {
	config := loadConfig()

	// default registry value
	if config.Registry == "" {
		config.Registry = "gcr.io"
	}

	// must use the fully qualified repo name. If the
	// repo name does not have the registry prefix we
	// should prepend.
	if !strings.HasPrefix(config.Repo, config.Registry) {
		config.Repo = path.Join(config.Registry, config.Repo)
	}

	os.Setenv("PLUGIN_REPO", config.Repo)
	os.Setenv("PLUGIN_REGISTRY", config.Registry)
	os.Setenv("DOCKER_USERNAME", config.Username)
	if config.AccessToken != "" {
		os.Setenv("ACCESS_TOKEN", config.AccessToken)
	} else {
		os.Setenv("DOCKER_PASSWORD", config.Password)
	}
	os.Setenv("PLUGIN_REGISTRY_TYPE", "GCR")

	// invoke the base docker plugin binary
	docker.Run()
}

func getenv(key ...string) (s string) {
	for _, k := range key {
		s = os.Getenv(k)
		if s != "" {
			return
		}
	}
	return
}
