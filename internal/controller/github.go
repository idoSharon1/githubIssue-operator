package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	assignmentcoreiov1 "github.com/idoSharon1/githubIssue-operator/api/v1"
	config "github.com/idoSharon1/githubIssue-operator/cmd/config"
	"github.com/idoSharon1/githubIssue-operator/internal/controller/utils"
)

var loadedConfig, _ = config.LoadConfig()
var restyClient = resty.New()

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

func (r *GithubIssueReconciler) openIssue(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue) error {
	logger := log.FromContext(ctx)
	owner, repo := r.extractRepoAndOwner(githubIssueInstance)

	res, err := restyClient.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(loadedConfig.EnvName))).
		SetBody(map[string]interface{}{
			"owner": owner,
			"repo":  repo,
			"title": githubIssueInstance.Spec.Title,
			"body":  githubIssueInstance.Spec.Description,
		}).
		Post(fmt.Sprintf("https://%s/repos/%s/%s/issues", loadedConfig.GithubApi.BaseUrl, owner, repo))

	if res.StatusCode() == 401 {
		logger.Error(err, "Bad credentials, please update the access token in your secret")
		return errors.New("bad credentials")
	}

	if err != nil {
		logger.Error(err, "Could not create new issue at this point")
		return err
	}

	return nil
}

func (r *GithubIssueReconciler) findRelevantIssue(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue) (utils.GithubReponseWantedProperties, error) {
	logger := log.FromContext(ctx)
	allRepoIssues, err := r.getAllRepoIssues(ctx, githubIssueInstance)
	var foundIssue utils.GithubReponseWantedProperties

	if err != nil {
		return foundIssue, err
	}

	for _, currentRepoIssue := range allRepoIssues {
		if currentRepoIssue.Title == githubIssueInstance.Spec.Title {
			logger.Info("Found the wanted issue")
			foundIssue = currentRepoIssue
			break
		}
	}

	return foundIssue, nil
}

func (r *GithubIssueReconciler) updateIssueOnRepoIfNeeded(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue) error {
	logger := log.FromContext(ctx)

	issueOnRepo, err := r.findRelevantIssue(ctx, githubIssueInstance)

	if err != nil {
		return err
	}

	if issueOnRepo.Description != githubIssueInstance.Spec.Description {
		logger.Info(fmt.Sprintf("Trying to update issue %s value to %s", githubIssueInstance.Spec.Title, githubIssueInstance.Spec.Description))
		err := r.updateIssue(ctx, githubIssueInstance, issueOnRepo, utils.UpdatedValue{Key: "body", Value: githubIssueInstance.Spec.Description})

		if err != nil {
			return err
		}

		logger.Info("Updated Successfully")
	} else {
		logger.Info("No need to update remote issue")
	}

	return nil
}

func (r *GithubIssueReconciler) closeIssue(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue) error {
	logger := log.FromContext(ctx)

	logger.Info("Trying to close issue")

	issueOnRepo, err := r.findRelevantIssue(ctx, githubIssueInstance)

	if err != nil {
		logger.Error(err, "Could not get remote issue on repo")
		return err
	}

	err = r.updateIssue(ctx, githubIssueInstance, issueOnRepo, utils.UpdatedValue{Key: "state", Value: "closed"})

	if err != nil {
		logger.Error(err, "Could not change status of issue to closed")
		return err
	}

	return nil
}

func (r *GithubIssueReconciler) updateIssue(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue, remoteIssue utils.GithubReponseWantedProperties, updatedValue utils.UpdatedValue) error {
	logger := log.FromContext(ctx)
	owner, repo := r.extractRepoAndOwner(githubIssueInstance)

	res, err := restyClient.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(loadedConfig.EnvName))).
		SetBody(map[string]interface{}{
			"owner":          owner,
			"repo":           repo,
			"issue_number":   string(remoteIssue.Number),
			"title":          remoteIssue.Title,
			updatedValue.Key: updatedValue.Value,
		}).
		Post(fmt.Sprintf("https://%s/repos/%s/%s/issues/%s", loadedConfig.GithubApi.BaseUrl, owner, repo, strconv.Itoa(remoteIssue.Number)))

	if res.StatusCode() == 401 {
		logger.Error(err, "Bad credentials, please update the access token in your secret")
		return errors.New("bad credentials")
	}

	if err != nil {
		logger.Error(err, "Failed to update remote issue")
		return err
	}

	return nil
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

func (r *GithubIssueReconciler) isIssueExist(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue) (bool, error) {
	logger := log.FromContext(ctx)
	isExist := false

	allRepoIssues, err := r.getAllRepoIssues(ctx, githubIssueInstance)

	if err != nil {
		return isExist, err
	}

	for _, currentRepoIssue := range allRepoIssues {
		if currentRepoIssue.Title == githubIssueInstance.Spec.Title {
			logger.Info(fmt.Sprintf("Issues with title -> %s already exist on this repo", githubIssueInstance.Spec.Title))
			isExist = true
			break
		}
	}

	return isExist, nil
}

func (r *GithubIssueReconciler) getAllRepoIssues(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue) ([]utils.GithubReponseWantedProperties, error) {
	logger := log.FromContext(ctx)
	owner, repo := r.extractRepoAndOwner(githubIssueInstance)
	var githubIssues []utils.GithubReponseWantedProperties

	res, err := restyClient.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(loadedConfig.EnvName))).
		SetResult(&githubIssues).
		Get(fmt.Sprintf("https://%s/repos/%s/%s/issues", loadedConfig.GithubApi.BaseUrl, owner, repo))

	if res.StatusCode() == 401 {
		logger.Error(err, "Bad credentials, please update the access token in your secret")
		return nil, errors.New("bad credentials")
	}

	if err != nil {
		logger.Error(err, "Could not list all the issues of the wanted repository")

		return nil, err
	}

	return githubIssues, nil
}
