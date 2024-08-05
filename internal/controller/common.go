package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	assignmentcoreiov1 "github.com/idoSharon1/githubIssue-operator/api/v1"
	utils "github.com/idoSharon1/githubIssue-operator/internal/controller/utils"
)

func (r *GithubIssueReconciler) ensureSecret(githubIssueInstance *assignmentcoreiov1.GithubIssue, ctx context.Context, req ctrl.Request, githubSecretName string, githubSecretKeyName string) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)

	githubSecret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: githubSecretName}, githubSecret)

	if err != nil {
		if errors.IsNotFound(err) {
			// Create default secret (user will have to update to his access token).
			logger.Info(fmt.Sprintf("Creating default secret at namespace -> %s", req.Namespace))
			err := r.Create(ctx, r.GithubDefaultAuthSecret(githubIssueInstance, types.NamespacedName{Namespace: req.Namespace, Name: githubSecretName}, githubSecretKeyName))

			if err != nil {
				logger.Error(err, "Error at creating default secret, requeue reconcile function")
				return &ctrl.Result{}, err
			} else {
				logger.Info("Successfully created default secret")
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

func (r *GithubIssueReconciler) containsCondition(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue, conditionValue string) bool {

	output := false
	for _, condition := range githubIssueInstance.Status.Conditons {
		if condition.Reason == conditionValue {
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

func (r *GithubIssueReconciler) setCondition(ctx context.Context, githubIssueInstance *assignmentcoreiov1.GithubIssue, typeName string, status metav1.ConditionStatus, reason string, message string) {
	logger := log.FromContext(ctx)

	if !r.containsCondition(ctx, githubIssueInstance, reason) {
		err := r.appendCondition(ctx, githubIssueInstance, typeName, status, reason, message)

		if err != nil {
			logger.Error(err, "Could not update status")
		}
	}
}
