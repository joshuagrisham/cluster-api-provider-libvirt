package controller

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/klog/v2"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	clog "sigs.k8s.io/cluster-api/util/log"
	"sigs.k8s.io/cluster-api/util/patch"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/joshuagrisham/cluster-api-provider-libvirt/api/v1beta2"
	"github.com/joshuagrisham/cluster-api-provider-libvirt/internal/libvirtclient"
)

// LibvirtMachineReconciler reconciles a LibvirtMachine object
type LibvirtMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=libvirtmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=libvirtmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=libvirtmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;machinesets;machines,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *LibvirtMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the LibvirtMachine instance
	libvirtMachine := &infrav1.LibvirtMachine{}
	if err := r.Get(ctx, req.NamespacedName, libvirtMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Get the LibvirtClientMachine instance
	externalMachine := getLibvirtClientMachine(libvirtMachine)

	// // If the LibvirtMachine is in an error state, return early.
	// if libvirtMachine.Status.FailureReason != nil || libvirtMachine.Status.FailureMessage != nil {
	// 	log.Info("Error state detected, skipping reconciliation")
	// 	return reconcile.Result{}, nil
	// }

	// Initialize patch helper early
	patchHelper, err := patch.NewHelper(libvirtMachine, r.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Always patch at the end
	defer func() {
		if err := patchHelper.Patch(ctx, libvirtMachine); err != nil {
			log.Error(err, fmt.Sprintf("failed to patch LibvirtMachine %s/%s", libvirtMachine.Namespace, libvirtMachine.Name))
			if rerr == nil {
				rerr = err
			}
		}
	}()

	// If the LibvirtMachine doesn't have our finalizer, add it
	controllerutil.AddFinalizer(libvirtMachine, infrav1.MachineFinalizer)

	// Add the owners of LibvirtMachine as k/v pairs to the logger
	ctx, log, err = clog.AddOwners(ctx, r.Client, libvirtMachine)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Fetch the LibvirtMachine's Machine
	machine, err := util.GetOwnerMachine(ctx, r.Client, libvirtMachine.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if machine == nil {
		log.Info(fmt.Sprintf("waiting for machine controller to set OwnerRef on LibvirtMachine %s/%s", libvirtMachine.Namespace, libvirtMachine.Name))
		return reconcile.Result{}, nil
	}

	// Fetch the Machine's Cluster
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info(fmt.Sprintf("LibvirtMachine %s/%s owner Machine is missing cluster label or cluster does not exist", libvirtMachine.Namespace, libvirtMachine.Name))
		return reconcile.Result{}, err
	}
	if cluster == nil {
		log.Info(fmt.Sprintf("Please associate LibvirtMachine %s/%s with a cluster using the label %s: <name of cluster>", libvirtMachine.Namespace, libvirtMachine.Name, clusterv1.ClusterNameLabel))
		return reconcile.Result{}, nil
	}

	// Add Cluster name to logger
	log = log.WithValues("Cluster", klog.KObj(cluster))
	ctx = ctrl.LoggerInto(ctx, log)

	// Fetch the LibvirtCluster
	libvirtCluster := &infrav1.LibvirtCluster{}
	libvirtClusterName := client.ObjectKey{
		Namespace: libvirtMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Get(ctx, libvirtClusterName, libvirtCluster); err != nil {
		// Handle deletion of orphaned LibvirtMachines in case the LibvirtCluster is already deleted
		if !libvirtMachine.DeletionTimestamp.IsZero() {
			return deleteExternalMachine(ctx, libvirtMachine, externalMachine)
		}
		log.Info("LibvirtCluster is not available yet")
		return reconcile.Result{}, nil
	}

	// Add LibvirtCluster name to logger
	log = log.WithValues("LibvirtCluster", klog.KObj(libvirtCluster))
	ctx = ctrl.LoggerInto(ctx, log)

	// Do nothing if the Cluster or LibvirtCluster is paused
	if annotations.IsPaused(cluster, libvirtCluster) {
		log.Info(fmt.Sprintf("LibvirtCluster %s/%s or linked Cluster %s/%s is marked as paused. Won't reconcile", libvirtCluster.Namespace, libvirtCluster.Name, cluster.Namespace, cluster.Name))
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Handle deleted instances
	if !libvirtMachine.DeletionTimestamp.IsZero() {
		return deleteExternalMachine(ctx, libvirtMachine, externalMachine)
	}

	// Do nothing if the Cluster's infrastructureRef is not defined
	if !cluster.Spec.InfrastructureRef.IsDefined() {
		log.Info(fmt.Sprintf("Cluster %s/%s infrastructureRef is not available yet", cluster.Namespace, cluster.Name))
		return reconcile.Result{}, nil
	}

	// Do nothing if the Cluster is not yet marked as provisioned
	if cluster.Status.Initialization.InfrastructureProvisioned == nil || *cluster.Status.Initialization.InfrastructureProvisioned != true {
		log.Info(fmt.Sprintf("Cluster %s/%s is not provisioned yet", cluster.Namespace, cluster.Name))
		return reconcile.Result{}, nil
	}

	// Recreate the machine if it exists but is not reconciled
	if externalMachine.Exists() && !externalMachine.IsReconciled() {
		log.Info(fmt.Sprintf("destroying out-of-sync virtual machine '%s'", externalMachine.Name))
		if err := externalMachine.Destroy(); err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "failed to destroy out-of-sync virtual machine '%s'", externalMachine.Name)
		}
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Create the machine if it does not yet exist
	if !externalMachine.Exists() {

		// Make sure the bootstrap data secret is available and populated.
		if machine.Spec.Bootstrap.DataSecretName == nil {
			log.Info(fmt.Sprintf("waiting for the bootstrap provider controller to set bootstrap data for LibvirtMachine %s/%s", libvirtMachine.Namespace, libvirtMachine.Name))
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}

		// Get the bootstrap data
		bootstrapData, err := r.getBootstrapData(ctx, libvirtMachine.Namespace, *machine.Spec.Bootstrap.DataSecretName)
		if err != nil {
			return reconcile.Result{}, err
		}
		if bootstrapData == "" {
			log.Info(fmt.Sprintf("bootstrap data is not available yet for LibvirtMachine %s/%s", libvirtMachine.Namespace, libvirtMachine.Name))
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
		externalMachine.UserData = bootstrapData

		if err := externalMachine.Create(); err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "failed to create virtual machine '%s'", externalMachine.Name)
		}
		log.Info(fmt.Sprintf("creating virtual machine '%s'", externalMachine.Name))
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil

	}

	// Now that the machine exists, update the LibvirtMachine resource per the Cluster API contract

	libvirtMachine.Spec.ProviderID = fmt.Sprintf("libvirt:///%s", externalMachine.Name)

	// Check if the machine is ready (running)
	if !externalMachine.IsReady() {
		// Machine exists and is reconciled but not yet ready - requeue to check again
		log.Info(fmt.Sprintf("waiting for virtual machine '%s' to become ready", externalMachine.Name))
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Update the LibvirtMachine status with the VM's IP addresses
	addresses, err := externalMachine.GetIPAddresses()
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to get IP addresses for virtual machine '%s'", externalMachine.Name)
	}
	if len(addresses) == 0 {
		log.Info(fmt.Sprintf("waiting for IP address to be assigned to virtual machine '%s'", externalMachine.Name))
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
	var machineAddresses []clusterv1.MachineAddress
	for _, address := range addresses {
		machineAddresses = append(machineAddresses, clusterv1.MachineAddress{
			Type:    clusterv1.MachineExternalIP, // Assume that all addresses are external? (since the host should be bridged to the libvirt network and can access them)
			Address: address,
		})
	}
	if !slices.Equal(libvirtMachine.Status.Addresses, machineAddresses) {
		log.Info(fmt.Sprintf("got IP addresses for virtual machine '%s': %v", externalMachine.Name, addresses))
	}
	libvirtMachine.Status.Addresses = machineAddresses

	// Mark the LibvirtMachine as "provisioned"
	if !libvirtMachine.Status.Initialization.Provisioned {
		log.Info(fmt.Sprintf("LibvirtMachine %s/%s is provisioned", libvirtMachine.Namespace, libvirtMachine.Name))
	}
	libvirtMachine.Status.Ready = true                      // v1beta1
	libvirtMachine.Status.Initialization.Provisioned = true // v1beta2

	// TODO: Per the Cluster API contract, we SHOULD also set Conditions here as well.

	// Requeue to check every 5 minutes to handle drift just in case the VM goes down or gets modified
	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil

}

// SetupWithManager sets up the controller with the Manager
func (r *LibvirtMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.LibvirtMachine{}).
		Named("libvirtmachine").
		Complete(r)
}

// deleteExternalMachine handles deletion of the externalMachine and its associated resouces (volumes, etc)
func deleteExternalMachine(ctx context.Context, libvirtMachine *infrav1.LibvirtMachine, externalMachine *libvirtclient.LibvirtClientMachine) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if externalMachine.Exists() {
		log.Info(fmt.Sprintf("deleting virtual machine '%s'", externalMachine.Name))
		if err := externalMachine.Destroy(); err != nil {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, errors.Wrap(err, "failed to destroy LibvirtMachine")
		}
	}
	log.Info(fmt.Sprintf("deleting LibvirtMachine %s/%s", libvirtMachine.Namespace, libvirtMachine.Name))
	controllerutil.RemoveFinalizer(libvirtMachine, infrav1.MachineFinalizer)

	return reconcile.Result{}, nil
}

// getBootstrapData retrieves and returns the bootstrap data from the specified secret
func (r *LibvirtMachineReconciler) getBootstrapData(ctx context.Context, namespace string, dataSecretName string) (string, error) {
	s := &corev1.Secret{}
	key := client.ObjectKey{Namespace: namespace, Name: dataSecretName}
	if err := r.Get(ctx, key, s); err != nil {
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret '%s'", dataSecretName)
	}

	value, ok := s.Data["value"]
	if !ok {
		return "", errors.New(fmt.Sprintf("error retrieving bootstrap data: secret '%s' is missing the 'value' key", dataSecretName))
	}
	valueString := string(value)

	format, ok := s.Data["format"]
	formatString := string(format)
	if !ok {
		formatString = "cloud-config"
	}
	if formatString != "cloud-config" {
		return "", errors.Errorf("unsupported bootstrap data format: %s", formatString)
	}

	return valueString, nil
}

// getLibvirtClientMachine gets a new LibvirtClientMachine instance from a LibvirtMachine
func getLibvirtClientMachine(libvirtMachine *infrav1.LibvirtMachine) *libvirtclient.LibvirtClientMachine {
	networkName := "default"
	if libvirtMachine.Spec.Network != nil {
		networkName = *libvirtMachine.Spec.Network
	}

	storagePoolName := "default"
	if libvirtMachine.Spec.StoragePool != nil {
		storagePoolName = *libvirtMachine.Spec.StoragePool
	}

	backingImageFormat := "qcow2"
	if libvirtMachine.Spec.BackingImageFormat != nil {
		backingImageFormat = *libvirtMachine.Spec.BackingImageFormat
	}

	return &libvirtclient.LibvirtClientMachine{
		// NOTE: ideally, we could use "{namespace}-{name}" like this: fmt.Sprintf("%s-%s", libvirtMachine.Namespace, libvirtMachine.Name)
		// but because this will become the hostname of the VM, this name can be too long in some cases (e.g. when created as part of a ClusterClass)
		Name:               libvirtMachine.Name,
		NetworkName:        networkName,
		StoragePoolName:    storagePoolName,
		CPU:                libvirtMachine.Spec.CPU,
		Memory:             libvirtMachine.Spec.Memory,
		DiskSize:           libvirtMachine.Spec.DiskSize,
		BackingImagePath:   libvirtMachine.Spec.BackingImagePath,
		BackingImageFormat: backingImageFormat,
	}
}
