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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	multitenancyv1 "codereliant.io/tenant/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=multitenancy.codereliant.io,resources=tenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=multitenancy.codereliant.io,resources=tenants/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=multitenancy.codereliant.io,resources=tenants/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Tenant object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
// +kubebuilder:rbac:groups=multitenancy.codereliant.io,resources=*,verbs=*
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=*
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=*,verbs=*
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	tenant := &multitenancyv1.Tenant{}

	log.Info("Reconciling tenant")

	// fetch the tenant instance
	if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
		// Tenant object not found, it might have been deleted
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Loop through each namespace defined in the Tenant Spec
	// Ensure the namespace exists, and if not, create it
	// Then ensure RoleBindings for each namespace
	for _, ns := range tenant.Spec.Namespaces {
		log.Info("Ensuring Namespace", "namespace", ns)
		if err := r.ensureNamespace(ctx, tenant, ns); err != nil {
			log.Error(err, "unable to ensure Namespace", "namespace", ns)
			return ctrl.Result{}, err
		}

		// for each ns ensuring rolebinding
		log.Info("Ensuring Admin RoleBinding", "namespace", ns)
		if err := r.EnsureRoleBinding(ctx, ns, tenant.Spec.AdminGroups, "admin"); err != nil {
			log.Error(err, "unable to ensure Admin RoleBinding", "namespace", ns)
			return ctrl.Result{}, err
		}

		log.Info("Ensuring User RoleBinding", "namespace", ns)
		if err := r.EnsureRoleBinding(ctx, ns, tenant.Spec.UserGroups, "user"); err != nil {
			log.Error(err, "unable to ensure User RoleBinding", "namespace", ns)
			return ctrl.Result{}, err
		}
	}

	// // Update the Tenant status with the current state
	tenant.Status.NamespaceCount = len(tenant.Spec.Namespaces)
	tenant.Status.AdminEmail = tenant.Spec.AdminEmail
	if err := r.Status().Update(ctx, tenant); err != nil {
		log.Error(err, "unable to update tenant status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multitenancyv1.Tenant{}).
		Complete(r)
}

const (
	tenantOperatorAnnotation = "tenant-operator"
)

func (r *TenantReconciler) ensureNamespace(ctx context.Context, tenant *multitenancyv1.Tenant, namespaceName string) error {
	log := log.FromContext(ctx)

	namespace := &v1.Namespace{}

	err := r.Get(ctx, client.ObjectKey{Name: namespaceName}, namespace)
	if err != nil {
		// If the namespace doesn't exist, create it
		if errors.IsNotFound(err) {
			log.Info("Creating Namespace", "namespace", namespaceName)
			namespace := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
					Annotations: map[string]string{
						"adminEmail": tenant.Spec.AdminEmail,
						"managed-by": tenantOperatorAnnotation,
					},
				},
			}

			// Attempt to create the namespace
			if err = r.Create(ctx, namespace); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// If the namespace already exists, check for required annotations
		log.Info("Namespace already exists", "namespace", namespaceName)

		// Logic for checking annotations
	}

	return nil
}

func (r *TenantReconciler) EnsureRoleBinding(ctx context.Context, namespaceName string, groups []string, clusterRoleName string) error {
	// roleBinding management implementation
	//log := log.FromContext(ctx)
	//

	return nil
}
