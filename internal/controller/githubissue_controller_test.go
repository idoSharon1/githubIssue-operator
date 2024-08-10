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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	assignmentcoreiov1 "github.com/idoSharon1/githubIssue-operator/api/v1"
)

var _ = Describe("GithubIssue Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}

		githubissue := &assignmentcoreiov1.GithubIssue{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind GithubIssue")
			err := k8sClient.Get(ctx, typeNamespacedName, githubissue)
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
			}
			correspondsSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: typeNamespacedName.Namespace, Name: fmt.Sprintf("%s-%s", typeNamespacedName.Name, loadedConfig.AuthSecret.GithubSecretName)}, correspondsSecret)
			if err != nil && errors.IsNotFound(err) {
				correspondsSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-%s", typeNamespacedName.Name, loadedConfig.AuthSecret.GithubSecretName),
						Namespace: typeNamespacedName.Namespace,
					},
					StringData: map[string]string{},
				}
				Expect(k8sClient.Create(ctx, correspondsSecret)).To(Succeed())
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
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
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

				resource.Spec.Title = "newTitle"
				err = k8sClient.Update(ctx, resource)
				Expect(err).NotTo(HaveOccurred())
				Eventually(func() bool {
					isSucceed, _ := controllerReconciler.isIssueExist(ctx, resource)

					return isSucceed
				}).Should(BeTrue())
			})
		})
	})
})
