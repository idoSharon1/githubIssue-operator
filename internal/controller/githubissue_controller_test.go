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

package controller

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	assignmentcoreiov1 "github.com/idoSharon1/githubIssue-operator/api/v1"
	"github.com/idoSharon1/githubIssue-operator/cmd/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("GithubIssue Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		githubissue := &assignmentcoreiov1.GithubIssue{}
		correspondsSecret := &corev1.Secret{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind GithubIssue")

			loadedConfig, tempErr := config.LoadConfig()
			Expect(tempErr).NotTo(HaveOccurred())

			err := k8sClient.Get(ctx, types.NamespacedName{Namespace: typeNamespacedName.Namespace, Name: fmt.Sprintf("%s-%s", typeNamespacedName.Name, loadedConfig.AuthSecret.GithubSecretName)}, correspondsSecret)
			if err != nil && errors.IsNotFound(err) {
				correspondsSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s", typeNamespacedName.Name, loadedConfig.AuthSecret.GithubSecretName),
						Namespace: typeNamespacedName.Namespace,
					},
					StringData: map[string]string{loadedConfig.AuthSecret.GithubSecretKeyName: os.Getenv("TESTING_ACCESS_TOKEN")},
				}
				Expect(k8sClient.Create(ctx, correspondsSecret)).To(Succeed())
				createdSecret := &corev1.Secret{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: correspondsSecret.Namespace, Name: correspondsSecret.Name}, createdSecret)
					return err == nil
				}).Should(BeTrue())
			}

			err = k8sClient.Get(ctx, typeNamespacedName, githubissue)
			if err != nil && errors.IsNotFound(err) {
				resource := &assignmentcoreiov1.GithubIssue{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
						Labels:    map[string]string{"test": "test"},
					},
					Spec: assignmentcoreiov1.GithubIssueSpec{
						Repo:        "https://github.com/idoSharon1/NamespaceLabel-operator",
						Title:       "test",
						Description: "test",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				createdGithubIssue := &assignmentcoreiov1.GithubIssue{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}, createdGithubIssue)
					return err == nil
				}).Should(BeTrue())
			}
		})

		AfterEach(func() {
			controllerReconciler := &GithubIssueReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			affectedResource := &assignmentcoreiov1.GithubIssue{}
			err := k8sClient.Get(ctx, typeNamespacedName, affectedResource)
			if err == nil {
				err := k8sClient.Delete(ctx, affectedResource)
				Expect(err).ToNot(HaveOccurred())
				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: affectedResource.Namespace, Name: affectedResource.Name}})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					deletedResource := &assignmentcoreiov1.GithubIssue{}
					err := k8sClient.Get(ctx, typeNamespacedName, deletedResource)
					return errors.IsNotFound(err)
				}).Should(BeTrue())
			}
		})

		It("should delete remote issue on delete", func() {
			By("implementing the finalizer logic", func() {
				controllerReconciler := &GithubIssueReconciler{
					Client: k8sClient,
					Scheme: k8sClient.Scheme(),
				}

				resource := &assignmentcoreiov1.GithubIssue{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}})
				Expect(err).NotTo(HaveOccurred())
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
				_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}})
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() bool {
					isSucceed, _ := controllerReconciler.isIssueExist(ctx, resource)

					return isSucceed
				}).Should(BeFalse())
			})
		})

		It("Should create remote issue if not exist", func() {
			By("running regular reconcile of new githubIssue", func() {
				controllerReconciler := &GithubIssueReconciler{
					Client: k8sClient,
					Scheme: k8sClient.Scheme(),
				}

				resource := &assignmentcoreiov1.GithubIssue{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())

				newTitle := "newTitle"
				resource.Spec.Title = newTitle
				err = k8sClient.Update(ctx, resource)
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() bool {
					updatedResource := &assignmentcoreiov1.GithubIssue{}
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}, updatedResource)

					if err != nil {
						return false
					}

					Expect(updatedResource.Spec.Title).To(Equal(newTitle))
					_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: updatedResource.Namespace, Name: updatedResource.Name}})
					Expect(err).NotTo(HaveOccurred())
					return true
				}).Should(BeTrue())
				Eventually(func() bool {

					isSucceed, _ := controllerReconciler.isIssueExist(ctx, resource)

					return isSucceed
				}).Should(BeTrue())
			})
		})

		It("Handle failed attempt to update remote issue", func() {
			By("Update the issue object status", func() {
				controllerReconciler := &GithubIssueReconciler{
					Client: k8sClient,
					Scheme: k8sClient.Scheme(),
				}

				resource := &assignmentcoreiov1.GithubIssue{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				newFailedRepo := resource.Spec.Repo + "test"
				resource.Spec.Repo = newFailedRepo
				err = k8sClient.Update(ctx, resource)
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() bool {
					updatedResource := &assignmentcoreiov1.GithubIssue{}
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}, updatedResource)
					Expect(err).NotTo(HaveOccurred())
					Expect(updatedResource.Spec.Repo).To(Equal(newFailedRepo))
					_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: updatedResource.Namespace, Name: updatedResource.Name}})
					Expect(err).To(HaveOccurred())
					err = k8sClient.Get(ctx, types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}, updatedResource)
					Expect(err).NotTo(HaveOccurred())
					return (controllerReconciler.containsCondition(updatedResource, "UserUpdatedHisAccessTokenInSecretAndExistingRepo", "False") || controllerReconciler.containsCondition(updatedResource, "IssueOpen", "False"))
				}).Should(BeTrue())

			})
		})

		It("Handle failed attemp to create remote issue", func() {
			By("Representing correct status of issue not open", func() {
				controllerReconciler := &GithubIssueReconciler{
					Client: k8sClient,
					Scheme: k8sClient.Scheme(),
				}
				resource := &assignmentcoreiov1.GithubIssue{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				correspondsSecret := &corev1.Secret{}
				err = k8sClient.Get(ctx, types.NamespacedName{Namespace: typeNamespacedName.Namespace, Name: fmt.Sprintf("%s-%s", typeNamespacedName.Name, loadedConfig.AuthSecret.GithubSecretName)}, correspondsSecret)
				Expect(err).ToNot(HaveOccurred())
				err = k8sClient.Delete(ctx, correspondsSecret)
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					createdResource := &assignmentcoreiov1.GithubIssue{}
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}, createdResource)
					Expect(err).NotTo(HaveOccurred())
					_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: createdResource.Namespace, Name: createdResource.Name}})
					Expect(err).To(HaveOccurred())
					err = k8sClient.Get(ctx, types.NamespacedName{Namespace: resource.Namespace, Name: resource.Name}, createdResource)
					Expect(err).NotTo(HaveOccurred())
					return (controllerReconciler.containsCondition(createdResource, "UserUpdatedHisAccessTokenInSecretAndExistingRepo", "False") || controllerReconciler.containsCondition(createdResource, "IssueOpen", "False"))
				}).Should(BeTrue())
			})
		})
	})
})
