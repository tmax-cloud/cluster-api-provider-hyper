/*


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

package controllers

import (
	"context"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infraexp "cluster-api-provider-hyper/apis/exp.infrastructure/v1alpha3"
	infrav1 "cluster-api-provider-hyper/apis/infrastructure/v1alpha3"
	infraUtil "cluster-api-provider-hyper/utils"
	"cluster-api-provider-hyper/utils/scope"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// HyperClusterReconciler reconciles a HyperCluster object
type HyperClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=hyperclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=hyperclusters/status,verbs=get;update;patch

func (r *HyperClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	log := r.Log.WithValues("hypercluster", req.NamespacedName)

	// your logic here
	// Get HyperCluster
	hc := &infrav1.HyperCluster{}

	if err := r.Get(ctx, req.NamespacedName, hc); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, hc.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}

	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	// Create the scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       r.Client,
		Logger:       log,
		Cluster:      cluster,
		HyperCluster: hc,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any HyperCluster changes.
	defer func() {
		if err := clusterScope.Close(); err != nil {
			log.Error(err, err.Error())
		}
	}()

	// Handle deleted clusters
	if !hc.DeletionTimestamp.IsZero() {
		return reconcileClusterDelete(clusterScope)
	}

	// Handle non-deleted clusters
	return reconcileClusterNormal(clusterScope)
}

func reconcileClusterDelete(clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func reconcileClusterNormal(clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := clusterScope.Logger
	if clusterScope.HyperCluster.Status.Ready {
		log.Info("apiEndpoint already exists. skip reconcile")
		return reconcile.Result{}, nil
	}

	apiEndpoint := getAPIEndpointfromHmp(clusterScope)
	if apiEndpoint == "" {
		log.Info("there is no proper resource in HyperMachinePool")
		return reconcile.Result{}, nil
	}

	clusterScope.HyperCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
		Host: apiEndpoint,
		Port: 6443,
	}
	clusterScope.HyperCluster.Status.Ready = true

	log.Info("apiEndpoint was allocated by " + apiEndpoint)
	return reconcile.Result{}, nil
}

// need to enhancement for getting apiEndpoint (ex. LB)
func getAPIEndpointfromHmp(clusterScope *scope.ClusterScope) string {
	hmpList := &infraexp.HyperMachinePoolList{}

	listOptions := []client.ListOption{
		client.MatchingLabels(map[string]string{infraUtil.LabelHyperMachinePoolValid: "true"}),
	}
	clusterScope.GetClient().List(context.TODO(), hmpList, listOptions...)

	if &hmpList.Items[0] == nil {
		return ""
	}

	oldhmp := &hmpList.Items[0]
	newhmp := oldhmp.DeepCopy()

	newhmp.Labels[infraUtil.LabelClusterRoleMaster] = ""

	clusterScope.GetClient().Patch(context.TODO(), newhmp, client.MergeFrom(oldhmp))

	return strings.Split(hmpList.Items[0].Spec.SSH.Address, ":")[0]
}

func (r *HyperClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.HyperCluster{}).
		WithEventFilter(
			predicate.Funcs{
				// Avoid reconciling if the event triggering the reconciliation is related to incremental status updates
				// for hypermachinepool resources only
				UpdateFunc: func(e event.UpdateEvent) bool {
					if e.ObjectOld.GetObjectKind().GroupVersionKind().Kind != "hypermachinepool" {
						return true
					}

					oldCluster := e.ObjectOld.(*infrav1.HyperCluster).DeepCopy()
					newCluster := e.ObjectNew.(*infrav1.HyperCluster).DeepCopy()

					oldCluster.Status = infrav1.HyperClusterStatus{}
					newCluster.Status = infrav1.HyperClusterStatus{}

					return !reflect.DeepEqual(oldCluster, newCluster)
				},
			},
		).
		Complete(r)
}
