package main

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	docker "github.com/drone-plugins/drone-buildx"
	"github.com/drone-plugins/drone-buildx/internal/gcp"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
)

type Config struct {
	Repo             string
	Registry         string
	Password         string
	WorkloadIdentity bool
	Username         string
	AccessToken      string
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
	repo := getenv("PLUGIN_REPO")
	registry := getenv("PLUGIN_REGISTRY")
	password := getenv(
		"PLUGIN_JSON_KEY",
		"GCR_JSON_KEY",
		"GOOGLE_CREDENTIALS",
		"TOKEN",
	)

	if idToken != "" && projectId != "" && poolId != "" && providerId != "" && serviceAccountEmail != "" {
		federalToken, err := gcp.GetFederalToken(idToken, projectId, poolId, providerId)
		if err != nil {
			logrus.Fatalf("Error (getFederalToken): %s", err)
		}
		accessToken, err := gcp.GetGoogleCloudAccessToken(federalToken, serviceAccountEmail)
		if err != nil {
			logrus.Fatalf("Error (getGoogleCloudAccessToken): %s", err)
		}
		config.AccessToken = accessToken
	} else {
		config.WorkloadIdentity = parseBoolOrDefault(false, getenv("PLUGIN_WORKLOAD_IDENTITY"))
		config.Username, config.Password = setUsernameAndPassword(username, password, config.WorkloadIdentity)
	}

	// default registry value
	if registry == "" {
		registry = "gcr.io"
	}

	// must use the fully qualified repo name. If the
	// repo name does not have the registry prefix we
	// should prepend.
	if !strings.HasPrefix(repo, registry) {
		repo = path.Join(registry, repo)
	}

	os.Setenv("PLUGIN_REPO", repo)
	os.Setenv("PLUGIN_REGISTRY", registry)
	os.Setenv("DOCKER_USERNAME", username)
	os.Setenv("DOCKER_PASSWORD", password)
	os.Setenv("PLUGIN_REGISTRY_TYPE", "GCR")

	return config
}

func main() {
	config := loadConfig()
	if config.AccessToken != "" {
		os.Setenv("ACCESS_TOKEN", config.AccessToken)
	} else if config.Username != "" && config.Password != "" {
		os.Setenv("DOCKER_USERNAME", config.Username)
		os.Setenv("DOCKER_PASSWORD", config.Password)
	}

	os.Setenv("PLUGIN_REPO", config.Repo)
	os.Setenv("PLUGIN_REGISTRY", config.Registry)
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

func getOauthToken(data []byte) (s string) {
	scopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
	}
	ctx := context.Background()
	credentials, err := google.CredentialsFromJSON(ctx, data, scopes...)
	if err == nil {
		token, err := credentials.TokenSource.Token()
		if err == nil {
			return token.AccessToken
		}
	}
	return
}

func parseBoolOrDefault(defaultValue bool, s string) (result bool) {
	var err error
	result, err = strconv.ParseBool(s)
	if err != nil {
		result = defaultValue
	}

	return
}

func setUsernameAndPassword(user string, pass string, workloadIdentity bool) (u string, p string) {
	// decode the token if base64 encoded
	decoded, err := base64.StdEncoding.DecodeString(pass)
	if err == nil {
		pass = string(decoded)
	}
	// get oauth token and set username if using workload identity
	if workloadIdentity {
		data := []byte(pass)
		pass = getOauthToken(data)
		user = "oauth2accesstoken"
	}
	return user, pass
}
