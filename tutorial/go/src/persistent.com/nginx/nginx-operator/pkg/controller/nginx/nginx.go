package nginx

import (
	"context"
	"time"

	nginxv1alpha1 "persistent.com/nginx/nginx-operator/pkg/apis/nginx/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const nginxPort = 80
//const nginxNodePort = 80
const nginxImage = "nginx:mainline-alpine-perl"

func nginxDeploymentName(v *nginxv1alpha1.Nginx) string {
	return v.Name + "-deployment"
}

func nginxServiceName(v *nginxv1alpha1.Nginx) string {
	return v.Name + "-service"
}

func (r *ReconcileNginx) nginxDeployment(v *nginxv1alpha1.Nginx) *appsv1.Deployment {
	labels := labels(v, "nginx")
	size := v.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:		nginxDeploymentName(v),
			Namespace: 	v.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:	nginxImage,
						ImagePullPolicy: corev1.PullAlways,
						Name:	"nginx-service",
						Ports:	[]corev1.ContainerPort{{
							ContainerPort: 	nginxPort,
							Name:			"nginx",
						}},
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(v, dep, r.scheme)
	return dep
}

func (r *ReconcileNginx) nginxService(v *nginxv1alpha1.Nginx) *corev1.Service {
	labels := labels(v, "nginx")

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:		nginxServiceName(v),
			Namespace: 	v.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol: corev1.ProtocolTCP,
				Port: nginxPort,
				TargetPort: intstr.FromInt(nginxPort),
				NodePort: 30685,
			}},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	controllerutil.SetControllerReference(v, s, r.scheme)
	return s
}

func (r *ReconcileNginx) updateNginxStatus(v *nginxv1alpha1.Nginx) (error) {
	//v.Status.BackendImage = nginxImage
	err := r.client.Status().Update(context.TODO(), v)
	return err
}

func (r *ReconcileNginx) handleNginxChanges(v *nginxv1alpha1.Nginx) (*reconcile.Result, error) {
	found := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Name:      nginxDeploymentName(v),
		Namespace: v.Namespace,
	}, found)
	if err != nil {
		// The deployment may not have been created yet, so requeue
		return &reconcile.Result{RequeueAfter:5 * time.Second}, err
	}

	size := v.Spec.Size

	if size != *found.Spec.Replicas {
		found.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			log.Error(err, "Failed to update Deployment.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return &reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return &reconcile.Result{Requeue: true}, nil
	}

	return nil, nil
}
