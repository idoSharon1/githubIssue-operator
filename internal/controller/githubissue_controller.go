package controller

import (
	"context"
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
			// reconcile function been called after the current github issue have been deleted.
			// TODO: handle finalizers
			return ctrl.Result{}, nil
		}

		// unexpected error happened requeue reconcile
		return ctrl.Result{}, err
	}

	var result *ctrl.Result

	// Ensure dependencies
	result, err = r.ensureSecret(instance, ctx, req, loadedConfig.AuthSecret.GithubSecretName, loadedConfig.AuthSecret.GithubSecretKeyName)
	if result != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	result, err = r.getValueFromSecretAndStoreEnv(ctx, req, loadedConfig.AuthSecret.GithubSecretName, loadedConfig.AuthSecret.GithubSecretKeyName, loadedConfig.EnvName)
	if result != nil {
		return ctrl.Result{}, err
	}

	r.addFinializersIfNeeded(instance, ctx)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GithubIssueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&assignmentcoreiov1.GithubIssue{}).
		Complete(r)
}
