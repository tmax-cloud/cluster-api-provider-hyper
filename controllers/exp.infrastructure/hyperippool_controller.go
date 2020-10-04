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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	expinfrav1 "cluster-api-provider-hyper/apis/exp.infrastructure/v1alpha3"
	infraUtil "cluster-api-provider-hyper/utils"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// HyperIPPoolReconciler reconciles a HyperIPPool object
type HyperIPPoolReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	patchHelper *patch.Helper
}

// +kubebuilder:rbac:groups=exp.infrastructure.cluster.x-k8s.io,resources=hyperippools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=exp.infrastructure.cluster.x-k8s.io,resources=hyperippools/status,verbs=get;update;patch

func (r *HyperIPPoolReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	_ = r.Log.WithValues("hyperippool", req.NamespacedName)

	// your logic here
	// get hip
	hip := &expinfrav1.HyperIPPool{}
	err := r.Get(ctx, req.NamespacedName, hip)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//set helper
	if helper, err := patch.NewHelper(hip, r.Client); err != nil {
		return ctrl.Result{}, err
	} else {
		r.patchHelper = helper
	}

	defer func() {
		r.Close(hip)
	}()

	if hip.Labels == nil {
		hip.Labels = map[string]string{
			infraUtil.LabelHyperIPPool: "",
		}
	} else if _, ok := hip.Labels[infraUtil.LabelHyperIPPool]; !ok {
		hip.Labels[infraUtil.LabelHyperIPPool] = ""
	}

	return ctrl.Result{}, nil
}

func (r *HyperIPPoolReconciler) Close(hip *expinfrav1.HyperIPPool) {
	r.patchHelper.Patch(context.TODO(), hip)
}

func (r *HyperIPPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&expinfrav1.HyperIPPool{}).
		WithEventFilter(
			predicate.Funcs{
				// Avoid reconciling if the event triggering the reconciliation is related to incremental status updates
				// for hyperippool resources only
				UpdateFunc: func(e event.UpdateEvent) bool {
					if e.ObjectOld.GetObjectKind().GroupVersionKind().Kind != "HyperIPPool" {
						return true
					}

					oldHip := e.ObjectOld.(*expinfrav1.HyperIPPool).DeepCopy()
					newHip := e.ObjectNew.(*expinfrav1.HyperIPPool).DeepCopy()

					return !reflect.DeepEqual(oldHip, newHip)
				},
			},
		).
		Complete(r)
}
