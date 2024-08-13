package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	assignmentcoreiov1 "github.com/idoSharon1/githubIssue-operator/api/v1"
	config "github.com/idoSharon1/githubIssue-operator/cmd/config"
)

// GithubIssueReconciler reconciles a GithubIssue object
type GithubIssueReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=assignment.core.io.assignment.core.io,resources=githubissues,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=assignment.core.io.assignment.core.io,resources=githubissues/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=assignment.core.io.assignment.core.io,resources=githubissues/finalizers,verbs=update

func (r *GithubIssueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Enter reconcile function")

	loadedConfig, err := config.LoadConfig()

	if err != nil {
		logger.Error(err, "Could not load config requeue reconcile")
		return ctrl.Result{}, err
	}

	instance := &assignmentcoreiov1.GithubIssue{}
	err = r.Client.Get(ctx, req.NamespacedName, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Item deleted successfully")
			return ctrl.Result{}, nil
		}

		// unexpected error happened requeue reconcile
		return ctrl.Result{}, err
	}

	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// This item has been marked for deletion
		if r.isFinalizerExist(instance) {
			err := r.closeIssue(ctx, instance)

			if err != nil {
				logger.Error(err, "Could not preform finalizer actions, this object will be deleted anyway")
			}

			err = r.removeFinalizer(instance, ctx)

			if err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	var result *ctrl.Result

	// Ensure dependencies
	result, err = r.ensureSecret(instance, ctx, req, loadedConfig.AuthSecret.GithubSecretName, loadedConfig.AuthSecret.GithubSecretKeyName)
	if result != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	result, err = r.getValueFromSecretAndStoreEnv(ctx, req, fmt.Sprintf("%s-%s", req.Name, loadedConfig.AuthSecret.GithubSecretName), loadedConfig.AuthSecret.GithubSecretKeyName, loadedConfig.EnvName)
	if result != nil {
		return ctrl.Result{}, err
	}

	r.addHelperLabelsIfNeeded(instance, ctx)
	r.addFinalizersIfNeeded(instance, ctx)

	existInRepo, err := r.isIssueExist(ctx, instance)

	if err != nil {
		logger.Error(err, "Could not verify if the issue is existing on repo")
		return ctrl.Result{}, err
	}

	if !existInRepo {
		err := r.openIssue(ctx, instance)

		if err != nil {
			r.setConditionIssueIsOpen(ctx, instance, "False")
			return ctrl.Result{}, err
		}

		r.setConditionIssueIsOpen(ctx, instance, "True")
	} else {
		isUpdated, err := r.updateIssueOnRepoIfNeeded(ctx, instance)

		if err != nil {
			return ctrl.Result{}, err
		}

		if isUpdated {
			r.updateConditionToAllOldRelevantObjects(ctx, instance)
		}
	}

	r.updateIssueHavePRCondition(ctx, instance)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubIssueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&assignmentcoreiov1.GithubIssue{}).
		Complete(r)
}
