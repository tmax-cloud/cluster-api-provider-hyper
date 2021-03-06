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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infraexp "cluster-api-provider-hyper/apis/exp.infrastructure/v1alpha3"
	infrav1 "cluster-api-provider-hyper/apis/infrastructure/v1alpha3"
	infraUtil "cluster-api-provider-hyper/utils"
	"cluster-api-provider-hyper/utils/scope"

	corev1 "k8s.io/api/core/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// HyperMachineReconciler reconciles a HyperMachine object
type HyperMachineReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=hypermachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=hypermachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch

func (r *HyperMachineReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	log := r.Log.WithValues("hypermachine", req.NamespacedName)

	// your logic here
	// Get HyperMachine
	hm := &infrav1.HyperMachine{}

	if err := r.Get(ctx, req.NamespacedName, hm); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !hm.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("finallizer handling!")
		return reconcileMachineDelete(hm, r.Client)
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, hm.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("machine", machine.Name)

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}

	log = log.WithValues("cluster", cluster.Name)

	// Fetch the HyperCluster
	hc := &infrav1.HyperCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.Spec.InfrastructureRef.Name, Namespace: cluster.Spec.InfrastructureRef.Namespace}, hc); err != nil {
		log.Info("Machine is missing hypercluster label or hypercluster does not exist")
		return ctrl.Result{}, err
	}

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Logger:       log,
		Client:       r.Client,
		Cluster:      cluster,
		Machine:      machine,
		HyperCluster: hc,
		HyperMachine: hm,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any HyperMachine changes.
	defer func() {
		if err := machineScope.Close(); err != nil {
			log.Error(err, err.Error())
		}
	}()

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(hm, infrav1.HyperMachineFinalizer) {
		controllerutil.AddFinalizer(hm, infrav1.HyperMachineFinalizer)
		return ctrl.Result{}, nil
	}

	return reconcileMachineNormal(machineScope)
}

func reconcileMachineNormal(machineScope *scope.MachineScope) (ctrl.Result, error) {
	if machineScope.HyperMachine.Status.Ready || *machineScope.HyperMachine.Spec.ProviderID != "" {
		machineScope.Logger.Info("HyperMachine's status ready")
		return ctrl.Result{}, nil
	}

	if machineScope.Machine.Status.Phase != "Provisioning" {
		machineScope.Logger.Info("Machine's phase isn't Provisioning ")
		return ctrl.Result{}, nil
	}

	//get hypermachinepool list for provisionig & doProvisioning
	if hmp := getHyperMachinePool(machineScope); hmp != nil {
		if cloudConfig, err := machineScope.GetBootstrapData(); err != nil {
			machineScope.Logger.Info("BootStrap Controller has not yet set bootstrap data")
			return ctrl.Result{}, nil
		} else {
			isMaster := false
			if _, ok := machineScope.Machine.Labels[infraUtil.LabelClusterRoleMaster]; ok {
				isMaster = true
			}
			if err, output := doProvisioning(machineScope, hmp, cloudConfig, isMaster); err == nil {
				//when doProvisioning succeeds
				machineScope.SetReady()
				machineScope.SetProviderID(hmp.Status.HostName)
				machineScope.SetAddress(strings.Split(hmp.Spec.SSH.Address, ":")[0])
			} else {
				machineScope.Logger.Error(err, output)
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func getHyperMachinePool(machineScope *scope.MachineScope) *infraexp.HyperMachinePool {
	hmp := &infraexp.HyperMachinePool{}
	hyperMachinePoolList := &infraexp.HyperMachinePoolList{}

	defer func() {
		adoptOrphan(machineScope, hmp)
	}()

	listOptions := []client.ListOption{
		client.MatchingLabels(map[string]string{
			infraUtil.LabelHyperMachinePoolValid: "true",
			infraUtil.LabelClusterRoleMaster:     "",
			infraUtil.LabelClusterName:           "",
		}),
	}
	machineScope.GetClient().List(context.TODO(), hyperMachinePoolList, listOptions...)

	if len(hyperMachinePoolList.Items) != 0 {
		hmp = &hyperMachinePoolList.Items[0]
		return &hyperMachinePoolList.Items[0]
	}

	listOptions = []client.ListOption{
		client.MatchingLabels(map[string]string{
			infraUtil.LabelHyperMachinePoolValid: "true",
			infraUtil.LabelClusterName:           "",
		}),
	}
	machineScope.GetClient().List(context.TODO(), hyperMachinePoolList, listOptions...)

	if len(hyperMachinePoolList.Items) != 0 {
		hmp = &hyperMachinePoolList.Items[0]
		return &hyperMachinePoolList.Items[0]
	}

	hmp = nil
	return nil
}

// adoptOrphan sets the MachineDeployment as a controller OwnerReference to the MachineSet.
func adoptOrphan(machinescope *scope.MachineScope, hmp *infraexp.HyperMachinePool) error {
	if hmp == nil {
		return nil
	}

	machinescope.HyperMachine.Spec.HyperMachinePoolRef = corev1.ObjectReference{
		APIVersion: hmp.APIVersion,
		Kind:       hmp.Kind,
		Name:       hmp.Name,
		Namespace:  hmp.Namespace,
		UID:        hmp.UID,
	}

	oldhmp := hmp.DeepCopy()
	newhmp := hmp.DeepCopy()

	newhmp.Labels[infraUtil.LabelClusterName] = machinescope.Cluster.Name

	return machinescope.GetClient().Patch(context.Background(), newhmp, client.MergeFrom(oldhmp))
}

func doProvisioning(machineScope *scope.MachineScope, hmp *infraexp.HyperMachinePool, value []byte, isMaster bool) (error, string) {
	switch hmp.Status.OS {
	case "ubuntu":
		return infraUtil.UbuntuProvisioning(machineScope, hmp, value, isMaster)
	case "centos":
		return infraUtil.CentosProvisioning(machineScope, hmp, value, isMaster)
	}

	return nil, ""
}

func reconcileMachineDelete(hm *infrav1.HyperMachine, c client.Client) (ctrl.Result, error) {
	//set helper
	if helper, err := patch.NewHelper(hm, c); err != nil {
		return ctrl.Result{}, err
	} else {
		doCleanHyperMachinePool(hm, c)
		controllerutil.RemoveFinalizer(hm, infrav1.HyperMachineFinalizer)

		helper.Patch(context.TODO(), hm)
	}

	return ctrl.Result{}, nil
}

func doCleanHyperMachinePool(hm *infrav1.HyperMachine, c client.Client) {
	ownRef := hm.Spec.HyperMachinePoolRef

	if &ownRef == nil || len(ownRef.Name) == 0 {
		return
	}

	hmp := &infraexp.HyperMachinePool{}
	key := types.NamespacedName{Namespace: ownRef.Namespace, Name: ownRef.Name}

	c.Get(context.TODO(), key, hmp)

	sshInfo := hmp.Spec.SSH

	conn, err := infraUtil.SSHConnect(sshInfo.Address, sshInfo.SSHid, sshInfo.SSHpw)
	if err != nil {
		return
	}

	conn.SendCommands("kubeadm reset --force")

	switch hmp.Status.OS {
	case "ubuntu":
		conn.SendCommands("apt-get purge -y keepalived")
	case "centos":
		conn.SendCommands("yum remove -y keepalived")
	}

	conn.Close()

	oldhmp := hmp.DeepCopy()
	newhmp := hmp.DeepCopy()

	newhmp.Labels[infraUtil.LabelClusterName] = ""
	if _, ok := newhmp.Labels[infraUtil.LabelClusterRoleMaster]; ok {
		delete(newhmp.Labels, infraUtil.LabelClusterRoleMaster)
	}

	c.Patch(context.TODO(), newhmp, client.MergeFrom(oldhmp))
}

func (r *HyperMachineReconciler) requeueHyperMachinesForBootStrapData(o handler.MapObject) []ctrl.Request {
	m := o.Object.(*clusterv1.Machine)
	log := r.Log.WithValues("objectMapper", "machineToHyperMachine", "namespace", m.Namespace, "machine", m.Name)

	// Don't handle deleted clusters
	if !m.ObjectMeta.DeletionTimestamp.IsZero() {
		log.V(4).Info("Machine has a deletion timestamp, skipping mapping.")
		return nil
	}

	// Make sure the ref is set
	if &m.Spec.InfrastructureRef == nil {
		log.V(4).Info("Machine does not have an InfrastructureRef, skipping mapping.")
		return nil
	}

	if m.Spec.InfrastructureRef.GroupVersionKind().Kind != "HyperMachine" {
		log.V(4).Info("Machine has an InfrastructureRef for a different type, skipping mapping.")
		return nil
	}

	hm := &infrav1.HyperMachine{}
	r.Get(context.TODO(), types.NamespacedName{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}, hm)
	if hm.Status.Ready {
		log.V(4).Info("HyperMachine has ready=true, skipping mapping.")
		return nil
	}

	log.V(4).Info("Adding request.", "HyperMachine", m.Spec.InfrastructureRef.Name)
	return []ctrl.Request{
		{
			NamespacedName: client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name},
		},
	}
}

func (r *HyperMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.HyperMachine{}).
		WithEventFilter(
			predicate.Funcs{
				// Avoid reconciling if the event triggering the reconciliation is related to incremental status updates
				// for hypermachinepool resources only
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldHyperMachine := e.ObjectOld.(*infrav1.HyperMachine).DeepCopy()
					newHyperMachine := e.ObjectNew.(*infrav1.HyperMachine).DeepCopy()

					oldHyperMachine.Status = infrav1.HyperMachineStatus{}
					newHyperMachine.Status = infrav1.HyperMachineStatus{}

					return !reflect.DeepEqual(oldHyperMachine, newHyperMachine)
				},
			},
		).
		Build(r)
	if err != nil {
		return err
	}

	return controller.Watch(
		&source.Kind{Type: &clusterv1.Machine{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(r.requeueHyperMachinesForBootStrapData),
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldMachine := e.ObjectOld.(*clusterv1.Machine)
				newMachine := e.ObjectNew.(*clusterv1.Machine)
				log := r.Log.WithValues("predicate", "updateEvent", "namespace", newMachine.Namespace, "machine", newMachine.Name)

				if oldMachine.Status.Phase == "Pending" && newMachine.Status.Phase == "Provisioning" {
					log.V(4).Info("Machine has Provisioning status.")
					return true
				}
				return false
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return false
			},
		},
	)
}
