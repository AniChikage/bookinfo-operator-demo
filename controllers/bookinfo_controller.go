/*
Copyright 2021.

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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	bookinfov1alpha1 "github.com/anichikage/bookinfo-operator/api/v1alpha1"
)

// BookinfoReconciler reconciles a Bookinfo object
type BookinfoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=bookinfo.demo.com,resources=bookinfoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bookinfo.demo.com,resources=bookinfoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bookinfo.demo.com,resources=bookinfoes/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *BookinfoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	bookinfo := &bookinfov1alpha1.Bookinfo{}
	err := r.Get(ctx, req.NamespacedName, bookinfo)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Bookinfo resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// log.Error(err, "Failed to get Bookinfo")
		// return ctrl.Result{}, err
	}

	// ======================   mysql ==========================

	// pvc
	pvc_mysql := r.pvcForMysql(bookinfo)
	pvc_mysql_found := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: pvc_mysql.Name, Namespace: bookinfo.Namespace}, pvc_mysql_found)
	if err != nil && errors.IsNotFound(err) {
		err = r.Create(ctx, pvc_mysql)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// deployment
	dep_sql := r.deploymentForBookInfoMysql(bookinfo)
	dep_sql_found := &appsv1.Deployment{}
	err = r.Get(context.TODO(), types.NamespacedName{
		Name:      dep_sql.Name,
		Namespace: bookinfo.Namespace,
	}, dep_sql_found)
	if err != nil && errors.IsNotFound(err) {
		err = r.Create(ctx, dep_sql)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep_sql.Namespace, "Deployment.Name", dep_sql.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	//svc
	svc_mysql_found := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: "mysql", Namespace: bookinfo.Namespace}, svc_mysql_found)
	if err != nil && errors.IsNotFound(err) {
		svc_mysql := r.serviceForMysql(bookinfo)
		err = r.Create(ctx, svc_mysql)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// ==================  mysql   end ============================

	// ====================   ratings   =====================

	// deployment
	dep_found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: "ratings", Namespace: bookinfo.Namespace}, dep_found)
	if err != nil && errors.IsNotFound(err) {
		dep := r.deploymentForBookInfoRatings(bookinfo)
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// service
	svc_found := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: "ratings", Namespace: bookinfo.Namespace}, svc_found)
	if err != nil && errors.IsNotFound(err) {
		svc := r.serviceForBookInfoRatings(bookinfo)
		err = r.Create(ctx, svc)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// replicas
	replicas := bookinfo.Spec.Replicas
	if *dep_found.Spec.Replicas != replicas {
		dep_found.Spec.Replicas = &replicas
		err = r.Update(ctx, dep_found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	podList := &corev1.PodList{}
	matchlabels := labelsForBookinfo(bookinfo.Name)
	listOpts := []client.ListOption{
		client.InNamespace(bookinfo.Namespace),
		client.MatchingLabels(matchlabels),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Bookinfo.Namespace", bookinfo.Namespace, "Bookinfo.Name", bookinfo.Name)
		return ctrl.Result{}, err
	}

	// Update status.Nodes if needed
	podNames := getPodNames(podList.Items)
	if !reflect.DeepEqual(podNames, bookinfo.Status.Nodes) {
		bookinfo.Status.Nodes = podNames
		err := r.Status().Update(ctx, bookinfo)
		if err != nil {
			log.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
	}

	// ==================  ratings   end ============================

	return ctrl.Result{}, nil
}

func (r *BookinfoReconciler) deploymentForBookInfoRatings(m *bookinfov1alpha1.Bookinfo) *appsv1.Deployment {
	replicas := m.Spec.Replicas
	dep_name := "ratings"
	matchlabels := labelsForBookinfo(m.Name)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ratings",
			Namespace: m.Namespace,
			// Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: matchlabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchlabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "docker.io/anichikage/ratings-v3-mysql:0.0.3",
						Name:  dep_name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 5000,
							Name:          "rate",
						}},
					}},
				},
			},
		},
	}

	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

func (r *BookinfoReconciler) serviceForBookInfoRatings(m *bookinfov1alpha1.Bookinfo) *corev1.Service {
	matchlabels := labelsForBookinfo(m.Name)

	svc := &corev1.Service{

		ObjectMeta: metav1.ObjectMeta{
			Name:      "ratings",
			Namespace: m.Namespace,
			// Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: matchlabels,
			Ports: []corev1.ServicePort{
				{
					Port: 5000,
					Name: "port",
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	ctrl.SetControllerReference(m, svc, r.Scheme)
	return svc
}

func (r *BookinfoReconciler) deploymentForBookInfoMysql(m *bookinfov1alpha1.Bookinfo) *appsv1.Deployment {
	dep_name := "mysql"
	matchlabels := labelsForBookinfoMysql(m.Name)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dep_name,
			Namespace: m.Namespace,
			// Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchlabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchlabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "mysql:5.6",
						Name:  dep_name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 3306,
							Name:          dep_name,
						}},
						Env: []corev1.EnvVar{
							{
								Name:  "MYSQL_ROOT_PASSWORD",
								Value: m.Spec.SQLRootPassword,
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "mysql-persistent-storage",
								MountPath: "/var/lib/mysql",
							},
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "mysql-persistent-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "mysql-pv-claim",
								},
							},
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

func (r *BookinfoReconciler) pvcForMysql(m *bookinfov1alpha1.Bookinfo) *corev1.PersistentVolumeClaim {
	labels := map[string]string{
		"app": m.Name,
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-pv-claim",
			Namespace: m.Namespace,
			Labels:    labels,
		},

		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: resource.MustParse("5Gi"),
				},
			},
		},
	}

	ctrl.SetControllerReference(m, pvc, r.Scheme)
	return pvc
}

func (r *BookinfoReconciler) serviceForMysql(m *bookinfov1alpha1.Bookinfo) *corev1.Service {
	matchlabels := labelsForBookinfoMysql(m.Name)

	svc := &corev1.Service{

		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql",
			Namespace: m.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: matchlabels,
			Ports: []corev1.ServicePort{
				{
					Port: 3306,
					Name: "port",
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	ctrl.SetControllerReference(m, svc, r.Scheme)
	return svc
}

func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func labelsForBookinfo(name string) map[string]string {
	return map[string]string{"app": "bookinfo", "bookinfo_cr": name}
}

func labelsForBookinfoMysql(name string) map[string]string {
	return map[string]string{"app": "bookinfo", "bookinfo_cr": name, "component": "mysql"}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BookinfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bookinfov1alpha1.Bookinfo{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
