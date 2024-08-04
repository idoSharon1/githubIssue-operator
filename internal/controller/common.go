package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
