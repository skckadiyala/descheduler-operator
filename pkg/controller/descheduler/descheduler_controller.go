package descheduler

import (
	"context"
	"fmt"
	"log"
	"strings"

	deschedulerv1alpha1 "github.com/skckadiyala/descheduler-operator/pkg/apis/descheduler/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var logfmt = logf.Log.WithName("controller_descheduler")

const (
	Running      = "RunningPhase"
	Updating     = "UpdatingPhase"
	DefaultImage = "skckadiyala/descheduler:v0.9.0"
)

// array of valid strategies
var validStrategies = []string{"duplicates", "interpodantiaffinity", "lownodeutilization", "nodeaffinity"}

// DeschedulerCommand provides descheduler command with policyconfigfile mounted as volume and log-level
var DeschedulerCommand = []string{"/bin/descheduler", "--policy-config-file", "/policy-dir/policy.yaml", "--v", "5"}

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Descheduler Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDescheduler{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("descheduler-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Descheduler
	err = c.Watch(&source.Kind{Type: &deschedulerv1alpha1.Descheduler{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Descheduler
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &deschedulerv1alpha1.Descheduler{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDescheduler implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDescheduler{}

// ReconcileDescheduler reconciles a Descheduler object
type ReconcileDescheduler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Descheduler object and makes changes based on the state read
// and what is in the Descheduler.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDescheduler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := logfmt.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Descheduler")

	// Fetch the Descheduler instance
	descheduler := &deschedulerv1alpha1.Descheduler{}
	err := r.client.Get(context.TODO(), request.NamespacedName, descheduler)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Descheduler %s/%s not found. Ignoring ", request.Namespace, request.Name)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Info("Failed to get descheduler %v", err)
		return reconcile.Result{}, err
	}

	//Descheduler. If descheduler object doesn't have any of the valid fields, return error
	// immediatly, don't proceed with config map/job creation

	if len(descheduler.Spec.Schedule) == 0 {
		reqLogger.Info("Deschdeular should have schedule for corn job set")
		return reconcile.Result{}, err
	}

	strategiesEnabled := getAllStrategiesEnabled(descheduler.Spec.Strategies)
	if err := validateStrategies(strategiesEnabled); err != nil {
		return reconcile.Result{}, err
	}

	// Generate Descheduler policy configmap

	if err := r.generateConfigMap(descheduler); err != nil {
		return reconcile.Result{}, err
	}

	if err := r.generateDeschedulerJob(descheduler); err != nil {
		return reconcile.Result{}, err
	}
	// TODO: Add validation logic to monitor the cronjob failed for n times
	// with image related issues(eg: ImagePullError etc) and create the cronjob
	// with default image specified above.

	if descheduler.Status.Phase != Running {
		if err := r.updateDeschedulerStatus(descheduler, Running); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

func getAllStrategiesEnabled(strategies []deschedulerv1alpha1.Strategy) []string {
	strategyName := make([]string, 0)
	for _, strategy := range strategies {
		strategyName = append(strategyName, strategy.Name)
	}
	return strategyName
}

func validateStrategies(strategies []string) error {
	if len(strategies) == 0 {
		err := fmt.Errorf("descheduler should have atleast one strategy enabled and if should be one of %v", strings.Join(validStrategies, ","))
		log.Printf("%v", err)
		return err
	}
	if len(strategies) > len(validStrategies) {
		err := fmt.Errorf("descheduler should have more strategyies enabled then supported %v", len(validStrategies))
		log.Printf("%v", err)
		return err
	}
	invalidStrategies := identifyInvalidStrategies(strategies)
	if len(invalidStrategies) > 0 {
		err := fmt.Errorf("expected one of the %v to be enabled but found following invalid strategies %v",
			strings.Join(validStrategies, ","), strings.Join(invalidStrategies, ","))

		log.Printf("%v", err)
		return err
	}
	return nil
}

func identifyInvalidStrategies(strategies []string) []string {
	invalidStrategiesEnabled := make([]string, 0)
	for _, strategy := range strategies {
		validStrategyFound := false
		for _, validStrategy := range validStrategies {
			if strings.ToUpper(strategy) == strings.ToUpper(validStrategy) {
				validStrategyFound = true
			}
		}
		if !validStrategyFound {
			invalidStrategiesEnabled = append(invalidStrategiesEnabled, strategy)
		}
	}
	return invalidStrategiesEnabled
}
