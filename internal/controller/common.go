package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	assignmentcoreiov1 "github.com/idoSharon1/githubIssue-operator/api/v1"
	utils "github.com/idoSharon1/githubIssue-operator/internal/controller/utils"
)

func (r *GithubIssueReconciler) ensureSecret(githubIssueInstance *assignmentcoreiov1.GithubIssue, ctx context.Context, req ctrl.Request, githubSecretName string, githubSecretKeyName string) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)

	githubSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: fmt.Sprintf("%s-%s", req.Name, githubSecretName)}, githubSecret)

	if err != nil {
		if errors.IsNotFound(err) {
			// Create default secret (user will have to update to his access token).
			logger.Info(fmt.Sprintf("Creating default secret at namespace -> %s", req.Namespace))
			err := r.Create(ctx, r.GithubDefaultAuthSecret(githubIssueInstance, types.NamespacedName{Namespace: req.Namespace, Name: fmt.Sprintf("%s-%s", req.Name, githubSecretName)}, githubSecretKeyName))

			if err != nil {
				logger.Error(err, "Error at creating default secret, requeue reconcile function")
				r.setCondition(ctx, githubIssueInstance, "AccessTokenSecretCreated", "False", "AccessTokenSecretCreated", "Error creatung github access token secret")
				return &ctrl.Result{}, err
			} else {
				logger.Info("Successfully created default secret")
				r.setCondition(ctx, githubIssueInstance, "AccessTokenSecretCreated", "True", "AccessTokenSecretCreated", "Successfully created github access token secret")
				return nil, nil
			}
		} else {
			logger.Error(err, "could not get secret, but the secret does exist")
			return &ctrl.Result{}, err
		}
	}

	return nil, nil
}

func (r *GithubIssueReconciler) getValueFromSecretAndStoreEnv(ctx context.Context, req ctrl.Request, secretName string, keyName string, envName string) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)

	wantedSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: secretName}, wantedSecret)

	if err != nil {
		if errors.IsNotFound(err) {
			logger.Error(err, "Secret is yet been created requeue")
		} else {
			logger.Error(err, fmt.Sprintf("Could not get secret when trying to find value for key %s in secret %s", keyName, secretName))
		}
		return &ctrl.Result{}, err
	} else {
		decodedString := string(wantedSecret.Data[keyName])

		err := utils.SetEnvironmentVariable(envName, decodedString)
		if err != nil {
			logger.Error(err, "Could not save value to env variable")
			return &ctrl.Result{}, err
		}

		return nil, nil
	}
}

func (r *GithubIssueReconciler) isFinalizerExist(githubIssueInstance *assignmentcoreiov1.GithubIssue) bool {
	return controllerutil.ContainsFinalizer(githubIssueInstance, loadedConfig.FinalizerKey)
}

func (r *GithubIssueReconciler) isHelpLabelsExist(githubIssueInstance *assignmentcoreiov1.GithubIssue) bool {
	if githubIssueInstance.GetLabels()[loadedConfig.TitleLabelKey] != "" && githubIssueInstance.GetLabels()[loadedConfig.RepoLabelKey] != "" {
		print("asfasf")
	}

	return githubIssueInstance.GetLabels()[loadedConfig.TitleLabelKey] != "" && githubIssueInstance.GetLabels()[loadedConfig.RepoLabelKey] != ""
}

func (r *GithubIssueReconciler) addHelperLabels(githubIssueInstance *assignmentcoreiov1.GithubIssue, ctx context.Context) error {
	githubIssueInstance.Labels[loadedConfig.TitleLabelKey] = githubIssueInstance.Spec.Title
	githubIssueInstance.Labels[loadedConfig.RepoLabelKey] = r.changeRepoToLabelFormat(githubIssueInstance)

	temp := r.changeRepoToLabelFormat(githubIssueInstance)
	print(temp)

	err := r.Update(ctx, githubIssueInstance)

	if err != nil {
		return err
	}

	return nil
}

func (r *GithubIssueReconciler) changeRepoToLabelFormat(githubIssueInstance *assignmentcoreiov1.GithubIssue) string {
	owner, repo := r.extractRepoAndOwner(githubIssueInstance)

	return fmt.Sprintf("%s.%s", owner, repo)
}

func (r *GithubIssueReconciler) addFinalizersIfNeeded(githubIssueInstance *assignmentcoreiov1.GithubIssue, ctx context.Context) {
	logger := log.FromContext(ctx)

	logger.Info("Checking if needing to add finalizers to current issue")

	if !r.isFinalizerExist(githubIssueInstance) {
		controllerutil.AddFinalizer(githubIssueInstance, loadedConfig.FinalizerKey)
		err := r.Update(ctx, githubIssueInstance)

		if err != nil {
			logger.Error(err, "Could not add finalizers this time, will try again next cycle")
		} else {
			logger.Info("Added finalizer")
		}
	} else {
		logger.Info("finalizer already exist")
	}
}

func (r *GithubIssueReconciler) addHelperLabelsIfNeeded(githubIssueInstance *assignmentcoreiov1.GithubIssue, ctx context.Context) {
	logger := log.FromContext(ctx)

	logger.Info("Checking if needing to add repo + title labels to current issue")

	if (!r.isHelpLabelsExist(githubIssueInstance)) ||
		(githubIssueInstance.GetLabels()[loadedConfig.RepoLabelKey] != r.changeRepoToLabelFormat(githubIssueInstance) ||
			githubIssueInstance.GetLabels()[loadedConfig.TitleLabelKey] != githubIssueInstance.Spec.Title) {
		logger.Info("Adding helper labels")
		err := r.addHelperLabels(githubIssueInstance, ctx)

		if err != nil {
			logger.Error(err, "Could not add helper labels, will stil continue lifecycle")
		}
	}
}

func (r *GithubIssueReconciler) removeFinalizer(githubIssueInstance *assignmentcoreiov1.GithubIssue, ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("Trying to remove finalizer from object")

	controllerutil.RemoveFinalizer(githubIssueInstance, loadedConfig.FinalizerKey)
	err := r.Update(ctx, githubIssueInstance)

	if err != nil {
		logger.Error(err, "Could not remove finalizer from item")
		return err
	}

	return nil
}

func (r *GithubIssueReconciler) containsCondition(githubIssueInstance *assignmentcoreiov1.GithubIssue, conditionValue string, newStatus metav1.ConditionStatus) bool {
	output := false

	for _, condition := range githubIssueInstance.Status.Conditons {
		if condition.Reason == conditionValue && condition.Status == newStatus {
			output = true
		}
	}
	return output
}

func (r *GithubIssueReconciler) appendCondition(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue,
	typeName string, status metav1.ConditionStatus, reason string, message string) error {

	log := log.FromContext(ctx)
	time := metav1.Time{Time: time.Now()}
	condition := metav1.Condition{Type: typeName, Status: status, Reason: reason, Message: message, LastTransitionTime: time}
	githubIssueInstance.Status.Conditons = append(githubIssueInstance.Status.Conditons, condition)

	err := r.Client.Status().Update(ctx, githubIssueInstance)
	if err != nil {
		log.Error(err, "GithubIssue resource status update failed.")
	}
	return nil
}

func (r *GithubIssueReconciler) setConditionIssueIsOpen(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue, status metav1.ConditionStatus) {
	const CONDITION_ISSUE_IS_OPEN_MESSAGE = "Issue opened successfully on github"
	const CONDITION_ISSUE_IS_OPEN_REASON = "IssueOpen"
	CONDITION_ISSUE_IS_OPEN_STATUS := status
	const CONDITION_ISSUE_IS_OPEN_TYPE = "IssueOpen"

	r.setCondition(ctx, githubIssueInstance, CONDITION_ISSUE_IS_OPEN_TYPE, CONDITION_ISSUE_IS_OPEN_STATUS, CONDITION_ISSUE_IS_OPEN_REASON, CONDITION_ISSUE_IS_OPEN_MESSAGE)
}

func (r *GithubIssueReconciler) setConditionIssueHasPullRequest(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue, status metav1.ConditionStatus) {
	const CONDITION_ISSUE_HAS_PR_MESSAGE = "Issue has pull request"
	const CONDITION_ISSUE_HAS_PR_REASON = "IssueHasPR"
	CONDITION_ISSUE_HAS_PR_STATUS := status
	const CONDITION_ISSUE_HAS_PR_TYPE = "IssueHasPR"

	r.setCondition(ctx, githubIssueInstance, CONDITION_ISSUE_HAS_PR_TYPE, CONDITION_ISSUE_HAS_PR_STATUS, CONDITION_ISSUE_HAS_PR_REASON, CONDITION_ISSUE_HAS_PR_MESSAGE)
}

func (r *GithubIssueReconciler) updateConditionToAllOldRelevantObjects(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue) {
	logger := log.FromContext(ctx)
	allIssues := &assignmentcoreiov1.GithubIssueList{}

	if r.isHelpLabelsExist(githubIssueInstance) {

		labelSelector := labels.Set{
			loadedConfig.RepoLabelKey:  r.changeRepoToLabelFormat(githubIssueInstance),
			loadedConfig.TitleLabelKey: githubIssueInstance.Spec.Title,
		}
		selector := labels.SelectorFromSet(labelSelector)

		listOptions := &client.ListOptions{
			LabelSelector: selector,
		}

		err := r.List(ctx, allIssues, client.InNamespace(""), listOptions)

		if err != nil {
			logger.Error(err, "Could not list all githubIssues in the cluster to update their status")
		}

		logger.Info("Update all relevant githubIssues in the cluster with their correspond status")
		for _, currentIssue := range allIssues.Items {
			if currentIssue.Spec.Description != githubIssueInstance.Spec.Description {
				r.setCondition(ctx, &currentIssue, "IssueDescriptionUnaffected", "True", "IssueDescriptionUnaffected", "Issue description not longer affected by this githubIssue")
			} else {
				r.setCondition(ctx, &currentIssue, "IssueDescriptionUnaffected", "False", "IssueDescriptionUnaffected", "Issue description not longer affected by this githubIssue")
			}
		}
	} else {
		logger.Info("Helper labels did not exist in this cycle could not update relevant status to all githubIssues")
	}
}

func (r *GithubIssueReconciler) setCondition(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue, typeName string, status metav1.ConditionStatus, reason string, message string) {
	logger := log.FromContext(ctx)

	if !r.containsCondition(githubIssueInstance, reason, status) {
		err := r.appendCondition(ctx, githubIssueInstance, typeName, status, reason, message)

		if err != nil {
			logger.Error(err, "Could not update status")
		}
	}
}
