package controller

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/klog/v2"

	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	clog "sigs.k8s.io/cluster-api/util/log"
	"sigs.k8s.io/cluster-api/util/patch"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/joshuagrisham/cluster-api-provider-libvirt/api/v1beta2"
)

// LibvirtClusterReconciler reconciles a LibvirtCluster object
type LibvirtClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=libvirtclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=libvirtclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=libvirtclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *LibvirtClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the LibvirtCluster instance
	libvirtCluster := &infrav1.LibvirtCluster{}
	if err := r.Get(ctx, req.NamespacedName, libvirtCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Initialize patch helper early
	patchHelper, err := patch.NewHelper(libvirtCluster, r.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Always patch at the end
	defer func() {
		if err := patchHelper.Patch(ctx, libvirtCluster); err != nil {
			log.Error(err, fmt.Sprintf("failed to patch LibvirtCluster %s/%s", libvirtCluster.Namespace, libvirtCluster.Name))
			if rerr == nil {
				rerr = err
			}
		}
	}()

	// If the LibvirtCluster doesn't have our finalizer, add it
	controllerutil.AddFinalizer(libvirtCluster, infrav1.MachineFinalizer)

	// Add the owners of LibvirtCluster as k/v pairs to the logger
	ctx, log, err = clog.AddOwners(ctx, r.Client, libvirtCluster)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Fetch the LibvirtCluster's Cluster
	cluster, err := util.GetOwnerCluster(ctx, r.Client, libvirtCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		log.Info(fmt.Sprintf("waiting for cluster controller to set OwnerRef on LibvirtCluster %s/%s", libvirtCluster.Namespace, libvirtCluster.Name))
		return reconcile.Result{}, nil
	}

	// Add Cluster name to logger
	log = log.WithValues("Cluster", klog.KObj(cluster))
	ctx = ctrl.LoggerInto(ctx, log)

	// Do nothing if the Cluster or LibvirtCluster is paused
	if annotations.IsPaused(cluster, libvirtCluster) {
		log.Info(fmt.Sprintf("LibvirtCluster %s/%s or linked Cluster %s/%s is marked as paused. Won't reconcile.", libvirtCluster.Namespace, libvirtCluster.Name, cluster.Namespace, cluster.Name))
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Handle deleted instances
	if !libvirtCluster.DeletionTimestamp.IsZero() {
		log.Info(fmt.Sprintf("deleting LibvirtCluster %s/%s", libvirtCluster.Namespace, libvirtCluster.Name))
		controllerutil.RemoveFinalizer(libvirtCluster, infrav1.MachineFinalizer)
		return reconcile.Result{}, nil
	}

	// Mark the LibvirtCluster as "provisioned"

	// TODO: Should we do potentially do any of the following first?
	// - Validate network exists (or create if not?)
	// - Validate storage pool exists (or create if not?)
	// - Set up some kind of load balancer VM e.g. HAProxy (if we want to provision a separate load balancer VM for the control plane endpoint?)

	libvirtCluster.Status.Ready = true                      // v1beta1
	libvirtCluster.Status.Initialization.Provisioned = true // v1beta2
	log.Info(fmt.Sprintf("LibvirtCluster %s/%s is provisioned", libvirtCluster.Namespace, libvirtCluster.Name))

	// TODO: Per the Cluster API contract, we SHOULD also set Conditions here as well.

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *LibvirtClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.LibvirtCluster{}).
		Named("libvirtcluster").
		Complete(r)
}
