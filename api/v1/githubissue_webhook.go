/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var githubissuelog = logf.Log.WithName("githubissue-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *GithubIssue) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
//+kubebuilder:webhook:path=/validate-assignment-core-io-assignment-core-io-v1-githubissue,mutating=false,failurePolicy=fail,sideEffects=None,groups=assignment.core.io.assignment.core.io,resources=githubissues,verbs=create;update,versions=v1,name=vgithubissue.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &GithubIssue{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *GithubIssue) ValidateCreate() (admission.Warnings, error) {
	githubissuelog.Info("validate create", "name", r.Name)

	return nil, r.validateRepoInputIsOk(r.Spec.Repo, field.NewPath("spec").Child("repo"))
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *GithubIssue) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	githubissuelog.Info("validate update", "name", r.Name)

	return nil, r.validateRepoInputIsOk(r.Spec.Repo, field.NewPath("spec").Child("repo"))
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *GithubIssue) ValidateDelete() (admission.Warnings, error) {
	githubissuelog.Info("validate delete", "name", r.Name)

	return nil, nil
}

func (r *GithubIssue) validateRepoInputIsOk(providedRepo string, fieldPath *field.Path) *field.Error {
	patternRegex := `^https:\/\/github\.com\/[\w-]+\/[\w-]+$`

	regex := regexp.MustCompile(patternRegex)

	if !regex.MatchString(providedRepo) {
		return field.Invalid(fieldPath, providedRepo, "The provided repo doesnt meet the github.com repo url requirments")
	}

	return nil
}
