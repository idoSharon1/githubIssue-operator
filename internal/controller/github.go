package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	assignmentcoreiov1 "github.com/idoSharon1/githubIssue-operator/api/v1"
	config "github.com/idoSharon1/githubIssue-operator/cmd/config"
)

var loadedConfig, _ = config.LoadConfig()

func (r *GithubIssueReconciler) GithubDefaultAuthSecret(githubIssueInstance *assignmentcoreiov1.GithubIssue, namespacedName types.NamespacedName, wantedTokenKey string) *corev1.Secret {
	defaultSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Type: "Opaque",
		StringData: map[string]string{
			wantedTokenKey: "{Insert your github access token here}",
		},
	}

	// Set the current github issue instance as the owner of this default secret (used fot automatic garbage collection).
	controllerutil.SetControllerReference(githubIssueInstance, defaultSecret, r.Scheme)
	return defaultSecret
}

func (r *GithubIssueReconciler) extractRepoAndOwner(githubIssueInstance *assignmentcoreiov1.GithubIssue) (owner string, repoName string) {
	givenUrl := githubIssueInstance.Spec.Repo
	urlSplitted := strings.Split(givenUrl, "/")
	lengthOfParts := len(urlSplitted)
	urlSplitted = urlSplitted[lengthOfParts-2:]

	// based of the github url standart of https://github.com/{owner}/{repo}
	owner = urlSplitted[0]
	repoName = urlSplitted[1]
	return
}

func (r *GithubIssueReconciler) getAllRepoIssues(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue) ([]assignmentcoreiov1.GithubIssueSpec, error) {
	logger := log.FromContext(ctx)
	owner, repo := r.extractRepoAndOwner(githubIssueInstance)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://%s/%s/%s/issues", loadedConfig.GithubApi.BaseUrl, owner, repo), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(loadedConfig.EnvName)))

	res, err := client.Do(req)

	if err != nil {
		logger.Error(err, "Could not list all the issues of the wanted repository")
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	fmt.Printf("Body: %s", body)

	return nil, nil
}
