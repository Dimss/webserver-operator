package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	routev1 "github.com/openshift/api/route/v1"

	oktov1alpha1 "github.com/webserver-operator/pkg/apis/okto/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_webserver")

type Sites struct {
	IndexCm string `json:"indexCm"`
	ConfCm  string `json:"confCm"`
}

// Add creates a new WebServer Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileWebServer{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("webserver-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource WebServer
	err = c.Watch(&source.Kind{Type: &oktov1alpha1.WebServer{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	// Watch for changes to Deployment
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &oktov1alpha1.WebServer{},
	})
	if err != nil {
		return err
	}
	// Watch for changes to Service
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &oktov1alpha1.WebServer{},
	})
	if err != nil {
		return err
	}
	// Watch for changes to ConfigMap
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &oktov1alpha1.WebServer{},
	})
	if err != nil {
		return err
	}
	// Watch for changes to Route
	err = c.Watch(&source.Kind{Type: &routev1.Route{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &oktov1alpha1.WebServer{},
	})
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileWebServer implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileWebServer{}

// ReconcileWebServer reconciles a WebServer object
type ReconcileWebServer struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileWebServer) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	sites := &[]Sites{}
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling WebServer")

	// Fetch the WebServer instance
	webServer := &oktov1alpha1.WebServer{}
	err := r.client.Get(context.TODO(), request.NamespacedName, webServer)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	//Check if deployment already exists, if not create a new one
	deployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: webServer.Name, Namespace: webServer.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		serverDeployment, err := r.deploymentForWebServer(webServer)
		if err != nil {
			reqLogger.Error(err, "error getting server deployment")
			return reconcile.Result{}, err
		}
		reqLogger.Info("Creating a new server deployment.", "Deployment.Namespace", serverDeployment.Namespace, "Deployment.Name", serverDeployment.Name)
		err = r.client.Create(context.TODO(), serverDeployment)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Server Deployment.", "Deployment.Namespace", serverDeployment.Namespace, "Deployment.Name", serverDeployment.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get server deployment.")
		return reconcile.Result{}, err
	}

	//Check if service already exists, if not create a new one
	service := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: webServer.Name, Namespace: webServer.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		serverService, err := r.serviceForWebServer(webServer)
		if err != nil {
			reqLogger.Error(err, "error getting server service")
			return reconcile.Result{}, err
		}
		reqLogger.Info("Creating a new service.", "Service.Namespace", serverService.Namespace, "Service.Name", serverService.Name)
		err = r.client.Create(context.TODO(), serverService)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Server Service.", "Service.Namespace", serverService.Namespace, "Service.Name", serverService.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get server service.")
		return reconcile.Result{}, err
	}

	//Check if route already exists, if not create a new one
	route := &routev1.Route{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: webServer.Name, Namespace: webServer.Namespace}, route)
	if err != nil && errors.IsNotFound(err) {
		serverRoute, err := r.routeForWebServer(webServer)
		if err != nil {
			reqLogger.Error(err, "error getting server route")
			return reconcile.Result{}, err
		}
		reqLogger.Info("Creating a new route.", "Route.Namespace", serverRoute.Namespace, "Router.Name", serverRoute.Name)
		err = r.client.Create(context.TODO(), serverRoute)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Server Route.", "Route.Namespace", serverRoute.Namespace, "Route.Name", serverRoute.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get server route.")
		return reconcile.Result{}, err
	}

	//Check if configmap for websites list already exists, if not create a new one
	cm := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: webServer.Spec.WebSitesRefCm, Namespace: webServer.Namespace}, cm)
	if err != nil && errors.IsNotFound(err) {
		websitesCm, err := r.configMapForWebServer(webServer)
		if err != nil {
			reqLogger.Error(err, "error getting websites ConfigMap")
			return reconcile.Result{}, err
		}
		reqLogger.Info("Creating a new cm.", "ConfigMap.Namespace", websitesCm.Namespace, "ConfigMap.Name", websitesCm.Name)
		err = r.client.Create(context.TODO(), websitesCm)
		if err != nil {
			reqLogger.Error(err, "Failed to create new ConfigMap.", "ConfigMap.Namespace", websitesCm.Namespace, "ConfigMap.Name", websitesCm.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get configmap.")
		return reconcile.Result{}, err
	} else {
		if err := r.syncSitesRefs(cm, sites); err != nil {
			reqLogger.Error(err, "Failed to sync sites ref cm.")
			return reconcile.Result{}, err
		}
		updateDeployment, err := r.syncSitesToWebServer(webServer, sites, deployment)
		if err != nil {
			reqLogger.Error(err, "Failed to add sites to deployment.")
			return reconcile.Result{}, err
		}
		// If update deployment true, run Deployment Update
		if updateDeployment {
			if err := r.client.Update(context.TODO(), deployment); err != nil {
				reqLogger.Error(err, "Failed to update deployment.")
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileWebServer) deploymentForWebServer(webServer *oktov1alpha1.WebServer) (*appsv1.Deployment, error) {
	var replicas int32
	replicas = 1
	labels := map[string]string{
		"app": webServer.Name,
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webServer.Name,
			Namespace: webServer.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": webServer.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   webServer.Name,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            webServer.Name,
							Image:           webServer.Spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(webServer, dep, r.scheme); err != nil {
		log.Error(err, "Error set controller reference for server deployment")
		return nil, err
	}
	return dep, nil
}

func (r *ReconcileWebServer) serviceForWebServer(webServer *oktov1alpha1.WebServer) (*corev1.Service, error) {
	labels := map[string]string{
		"app": webServer.Name,
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webServer.Name,
			Namespace: webServer.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": webServer.Name},
			Ports: []corev1.ServicePort{
				{
					Name: "https",
					Port: 8080,
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(webServer, service, r.scheme); err != nil {
		log.Error(err, "Error set controller reference for server service")
		return nil, err
	}
	return service, nil
}

func (r *ReconcileWebServer) routeForWebServer(webServer *oktov1alpha1.WebServer) (*routev1.Route, error) {
	labels := map[string]string{
		"app": webServer.Name,
	}
	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webServer.Name,
			Namespace: webServer.Namespace,
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationEdge,
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: webServer.Name,
			},
		},
	}
	if err := controllerutil.SetControllerReference(webServer, route, r.scheme); err != nil {
		log.Error(err, "Error set controller reference for server route")
		return nil, err
	}
	return route, nil
}

func (r *ReconcileWebServer) configMapForWebServer(webServer *oktov1alpha1.WebServer) (*corev1.ConfigMap, error) {
	labels := map[string]string{
		"app": webServer.Name,
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webServer.Spec.WebSitesRefCm,
			Namespace: webServer.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{"sites": "[]"},
	}
	//TODO: for the workshop
	if err := controllerutil.SetControllerReference(webServer, cm, r.scheme); err != nil {
		log.Error(err, "Error set controller reference for configmap")
		return nil, err
	}
	return cm, nil
}

func (r *ReconcileWebServer) syncSitesRefs(cm *corev1.ConfigMap, sites *[]Sites) error {
	if _, ok := cm.Data["sites"]; !ok {
		return fmt.Errorf("sites ref configmap is not valid, consider to delete the configmap")
	}
	if err := json.Unmarshal([]byte(cm.Data["sites"]), sites); err != nil {
		return fmt.Errorf("unable to unmarshal sites struct, consider to delete the configmap")
	}
	return nil
}

func (r *ReconcileWebServer) syncSitesToWebServer(
	webServer *oktov1alpha1.WebServer,
	sites *[]Sites,
	serverDeployment *appsv1.Deployment) (updateDeployment bool, err error) {

	updateDeployment = false
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	// Sync Deployment Volumes and Sites CM
	// If site exists in Deployment but the same site is missing from CM
	// remove that mount from Deployment
	for _, volume := range serverDeployment.Spec.Template.Spec.Volumes {
		siteName := volumeExistsInSites(sites, volume.Name)
		if siteName == nil {
			log.Info("No volumes found for", "siteName", siteName)
			updateDeployment = true
		} else {
			volumes = append(volumes, getVolume(*siteName))
		}
	}

	for _, container := range serverDeployment.Spec.Template.Spec.Containers {
		for _, volumeMount := range container.VolumeMounts {
			siteName, mountPath := volumeMountExistsInSites(sites, volumeMount.Name)
			if siteName == nil {
				updateDeployment = true
			} else {
				volumeMounts = append(volumeMounts, getVolumeMount(*siteName, *mountPath))
			}
		}
	}

	for _, site := range *sites {

		if !volumeExistsInDeployment(serverDeployment, site.ConfCm) {
			updateDeployment = true
			// Add Config CM
			volumes = append(volumes, getVolume(site.ConfCm))
		}

		if !volumeExistsInDeployment(serverDeployment, site.IndexCm) {
			updateDeployment = true
			// Add Config Index CM
			volumes = append(volumes, getVolume(site.IndexCm))
		}

	}
	for _, site := range *sites {
		if !volumeMountExistsInDeployment(serverDeployment, site.ConfCm) {
			updateDeployment = true
			// Add Config CM
			volumeMounts = append(volumeMounts, getVolumeMount(site.ConfCm, fmt.Sprintf("/opt/app-root/etc/nginx.d/%s", site.IndexCm)))
		}
		if !volumeMountExistsInDeployment(serverDeployment, site.IndexCm) {
			updateDeployment = true
			// Add Index CM
			volumeMounts = append(volumeMounts, getVolumeMount(site.IndexCm, fmt.Sprintf("/opt/app-root/src/%s", site.IndexCm)))

		}

	}

	serverDeployment.Spec.Template.Spec.Volumes = volumes
	webServerContainer := &serverDeployment.Spec.Template.Spec.Containers[0]
	webServerContainer.VolumeMounts = volumeMounts
	return updateDeployment, nil
}

func volumeExistsInDeployment(serverDeployment *appsv1.Deployment, volumeName string) bool {
	for _, volume := range serverDeployment.Spec.Template.Spec.Volumes {
		if volume.Name == volumeName {
			return true
		}
	}
	return false
}

func volumeMountExistsInDeployment(serverDeployment *appsv1.Deployment, volumeMountName string) bool {
	for _, container := range serverDeployment.Spec.Template.Spec.Containers {
		for _, volumeMount := range container.VolumeMounts {
			if volumeMount.Name == volumeMountName {
				return true
			}
		}
	}
	return false
}

func volumeExistsInSites(sites *[]Sites, volumeName string) *string {
	for _, site := range *sites {
		if site.ConfCm == volumeName {
			return &site.ConfCm
		}
		if site.IndexCm == volumeName {
			return &site.IndexCm
		}
	}
	return nil
}

func volumeMountExistsInSites(sites *[]Sites, volumeName string) (*string, *string) {
	confPath := "/opt/app-root/etc/nginx.d/%s"
	indexPath := "/opt/app-root/src/%s"
	for _, site := range *sites {
		if site.ConfCm == volumeName {
			mountPath := fmt.Sprintf(confPath, site.ConfCm)
			return &site.ConfCm, &mountPath
		}

		if site.IndexCm == volumeName {
			mountPath := fmt.Sprintf(indexPath, site.IndexCm)
			return &site.IndexCm, &mountPath
		}
	}
	return nil, nil
}

func getVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
			},
		},
	}
}

func getVolumeMount(name string, mount string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: mount,
	}
}
