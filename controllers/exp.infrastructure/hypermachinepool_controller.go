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
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	expinfrav1 "cluster-api-provider-hyper/apis/exp.infrastructure/v1alpha3"
	infraUtil "cluster-api-provider-hyper/utils"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// HyperMachinePoolReconciler reconciles a HyperMachinePool object
type HyperMachinePoolReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	patchHelper *patch.Helper
}

// +kubebuilder:rbac:groups=exp.infrastructure.cluster.x-k8s.io,resources=hypermachinepools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=exp.infrastructure.cluster.x-k8s.io,resources=hypermachinepools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *HyperMachinePoolReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	log := r.Log.WithValues("hypermachinepool", req.NamespacedName)

	// your logic here
	// get hmp
	hmp := &expinfrav1.HyperMachinePool{}
	err := r.Get(ctx, req.NamespacedName, hmp)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	//set helper
	if helper, err := patch.NewHelper(hmp, r.Client); err != nil {
		log.Error(err, "failed to init patch helper")
		return reconcile.Result{}, err
	} else {
		r.patchHelper = helper
	}

	defer func() {
		if err, errStr := r.Close(hmp); err != nil {
			log.Error(err, errStr)
		}
	}()

	// init
	if _, ok := hmp.Labels[infraUtil.LabelHyperMachinePoolValid]; !ok {
		initHyperMachinePool(hmp)
	}

	// check: sshInfo validation
	if ok, _ := strconv.ParseBool(hmp.Labels[infraUtil.LabelHyperMachinePoolValid]); !ok {
		validSSH(hmp)
	}

	return ctrl.Result{}, nil
}

func validSSH(hmp *expinfrav1.HyperMachinePool) {
	if ssh, err := infraUtil.SSHConnect(hmp.Spec.SSH.Address, hmp.Spec.SSH.SSHid, hmp.Spec.SSH.SSHpw); err != nil {
		hmp.Labels[infraUtil.LabelHyperMachinePoolValid] = "false"
		hmp.Status.Error = err.Error()
	} else {
		hmp.Labels[infraUtil.LabelHyperMachinePoolValid] = "true"

		//get os, kernel version
		var outStr string
		if out, err := ssh.SendCommands("uname -r"); err == nil {
			outStr = string(out)

			hmp.Status.Kernel = string(out)[:len(outStr)-2]
		}
		if out, err := ssh.SendCommands("cat /etc/*release | grep -w ID"); err == nil {
			outStr = string(out)

			hmp.Status.OS = string(out)[len(string("ID=")) : len(outStr)-2]
			if strings.Contains(hmp.Status.OS, "\"") {
				hmp.Status.OS = strings.Split(hmp.Status.OS, "\"")[1]
			}
		}
		if out, err := ssh.SendCommands("hostname"); err == nil {
			outStr = string(out)

			hmp.Status.HostName = string(out)[:len(outStr)-2]
		}
		switch hmp.Status.OS {
		case "ubuntu":
			if out, err := ssh.SendCommands("ifconfig | grep " + strings.Split(hmp.Spec.SSH.Address, ":")[0] + " -B 1"); err == nil {
				outStr = string(out)

				hmp.Status.NetworkInterface = strings.Split(outStr, ":")[0]
			}
		case "centos":
			if out, err := ssh.SendCommands("ip addr | grep " + strings.Split(hmp.Spec.SSH.Address, ":")[0] + " -B 3"); err == nil {
				outStr = string(out)
				hmp.Status.NetworkInterface = strings.Split(outStr, ":")[1][1:]
			}
		}
		ssh.Close()
	}
}

func initHyperMachinePool(hmp *expinfrav1.HyperMachinePool) bool {
	if hmp.Labels == nil {
		hmp.Labels = map[string]string{
			infraUtil.LabelHyperMachinePoolValid: "false",
			infraUtil.LabelClusterName:           "",
		}
		return true
	} else if _, ok := hmp.Labels[infraUtil.LabelHyperMachinePoolValid]; !ok {
		hmp.Labels[infraUtil.LabelHyperMachinePoolValid] = "false"
		hmp.Labels[infraUtil.LabelClusterName] = ""
		return true
	} else {
		return false
	}
}

func (r *HyperMachinePoolReconciler) Close(hmp *expinfrav1.HyperMachinePool) (error, string) {
	if err := r.patchHelper.Patch(context.TODO(), hmp); err != nil {
		return err, "failed to patch by patch helper"
	}
	return nil, ""
}

func (r *HyperMachinePoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&expinfrav1.HyperMachinePool{}).
		WithEventFilter(
			predicate.Funcs{
				// Avoid reconciling if the event triggering the reconciliation is related to incremental status updates
				// for hypermachinepool resources only
				UpdateFunc: func(e event.UpdateEvent) bool {
					if e.ObjectOld.GetObjectKind().GroupVersionKind().Kind != "HyperMachinePool" {
						return true
					}

					oldHmp := e.ObjectOld.(*expinfrav1.HyperMachinePool).DeepCopy()
					newHmp := e.ObjectNew.(*expinfrav1.HyperMachinePool).DeepCopy()

					oldHmp.Status = expinfrav1.HyperMachinePoolStatus{}
					newHmp.Status = expinfrav1.HyperMachinePoolStatus{}

					return !reflect.DeepEqual(oldHmp, newHmp)
				},
			},
		).
		Complete(r)
}
