/*
Copyright 2022.

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
	b64 "encoding/base64"
	"encoding/json"
	teamv1 "github.com/snapp-incubator/team-operator/api/v1"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	userv1 "github.com/openshift/api/user/v1"
)

var logf = log.Log.WithName("controller_team")

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}


//+kubebuilder:rbac:groups=team.snappcloud.io,resources=teams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=team.snappcloud.io,resources=teams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=team.snappcloud.io,resources=teams/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=user.openshift.io,resources=*,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the clus k8s.io/api closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Team object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *TeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	reqLogger := logf.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling team")
	team := &teamv1.Team{}

	err := r.Client.Get(context.TODO(), req.NamespacedName, team)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	} else {
		log.Info("team is found and teamAdmin is : " + team.Spec.TeamAdmin)

	}
	r.createArgocdStatiAdminUser(ctx, req)
	r.createArgocdStatiViewUser(ctx, req)

	return ctrl.Result{}, nil
}
func (r *TeamReconciler) createArgocdStatiAdminUser(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	reqLogger := logf.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling team")
	team := &teamv1.Team{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, team)

	log.Info("team is found and teamAdmin is : " + team.Spec.TeamAdmin)
	staticUser := map[string]map[string]string{
		"data": {
			"accounts." + req.Name+"-Admin-CI": "apiKey,login",
		},
	}
	staticUserByte, _ := json.Marshal(staticUser)
	err = r.Client.Patch(context.Background(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "argocd",
			Name:      "argocd-cm",
		},
	}, client.RawPatch(types.StrategicMergePatchType, staticUserByte))
	if err != nil {
		log.Error(err, "Failed to patch cm")
		return ctrl.Result{}, err
	}
	//set password to the user
	hash, _ := HashPassword(team.Spec.Argo.Admin.CIPass) // ignore error for the sake of simplicity

	encodedPass := b64.StdEncoding.EncodeToString([]byte(hash))
	staticPassword := map[string]map[string]string{
		"data": {
			"accounts." + req.Name+"-Admin-CI.password": encodedPass,
		},
	}
	staticPassByte, _ := json.Marshal(staticPassword)

	err = r.Client.Patch(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "argocd",
			Name:      "argocd-secret",
		},
	}, client.RawPatch(types.StrategicMergePatchType, staticPassByte))
	if err != nil {
		log.Error(err, "Failed to patch secret")
		return ctrl.Result{}, err
	}
	r.setRBACArgoCDAdminUser(ctx, req)

	return ctrl.Result{}, nil
}

func (r *TeamReconciler) createArgocdStatiViewUser(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	reqLogger := logf.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling team")
	team := &teamv1.Team{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, team)

	log.Info("team is found and teamAdmin is : " + team.Spec.TeamAdmin)
	staticUser := map[string]map[string]string{
		"data": {
			"accounts." + req.Name+"-View-CI": "apiKey,login",
		},
	}
	staticUserByte, _ := json.Marshal(staticUser)
	err = r.Client.Patch(context.Background(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "argocd",
			Name:      "argocd-cm",
		},
	}, client.RawPatch(types.StrategicMergePatchType, staticUserByte))
	if err != nil {
		log.Error(err, "Failed to patch cm")
		return ctrl.Result{}, err
	}
	//set password to the user
	hash, _ := HashPassword(team.Spec.Argo.View.CIPass) // ignore error for the sake of simplicity

	encodedPass := b64.StdEncoding.EncodeToString([]byte(hash))

	staticPassword := map[string]map[string]string{
		"data": {
			"accounts." + req.Name+"-View-CI" + ".password": encodedPass,
		},
	}
	staticPassByte, _ := json.Marshal(staticPassword)

	err = r.Client.Patch(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "argocd",
			Name:      "argocd-secret",
		},
	}, client.RawPatch(types.StrategicMergePatchType, staticPassByte))
	if err != nil {
		log.Error(err, "Failed to patch secret")
		return ctrl.Result{}, err
	}
	r.setRBACArgoCDViewUser(ctx, req)

	return ctrl.Result{}, nil
}
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func (r *TeamReconciler)setRBACArgoCDAdminUser(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	reqLogger := logf.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling team")
    team := &teamv1.Team{}
	found := &corev1.ConfigMap{}
	err1 := r.Client.Get(context.TODO(), req.NamespacedName, team)
	if err1 != nil {
		log.Error(err1, "Failed to get  team")
		return ctrl.Result{}, err1
	}
	err := r.Client.Get(ctx, types.NamespacedName{Name: "argocd-rbac-cm", Namespace: "argocd"}, found)
	if err != nil {
		log.Error(err, "Failed to get  cm")
		return ctrl.Result{}, err
	}
	//var policies []string
	// split := strings.Split(found.Data["policy.csv"], "\n")
	// for _, v := range split {
	// 	log.Info((v))
	// }
	log.Info("users")
	for _, user := range team.Spec.Argo.Admin.Users {
		log.Info(user)
	}
    group := &userv1.Group{}
   // grpname:= req.Name+"-admin"
	//err2 := r.Client.Get(context.Background(), grpname, group)
	//group, err := r.Client.Get(context.Background(), req.Name+"-admin", metav1.GetOptions{})
	//err2 := r.Client.Get(ctx, types.NamespacedName{Name: "test", Namespace: ""}, group)
	//err2 := r.Client.Get(ctx, &userv1.Group{ObjectMeta: metav1.ObjectMeta{Name: "test"}} , metav1.CreateOptions{})
	err2 := r.Client.Get(ctx, types.NamespacedName{Name:"test", Namespace: ""}, group)

	if err2 != nil{
		log.Error(err2,"Failed get group")
		return ctrl.Result{}, err
	}

	group.Users = append(group.Users, team.Spec.Argo.Admin.Users[1])
	err3 := r.Client.Update(ctx, group)
	if err3 != nil{
		log.Error(err3,"group doesnt exist")
		return ctrl.Result{}, err
	}

	// ocpGroup = &userv1.Group{
	// 	TypeMeta: metav1.TypeMeta{
	// 		Kind:       "Group",
	// 		APIVersion: userv1.GroupVersion.String(),
	// 	},
	// }

	log.Info("in setRBACArgoCDUser")
	log.Info(req.Name+"-Admin-CI")
	newPolicy :="g," +req.Name+"-Admin-CI,role: : " +req.Name+"-admin"
    duplicatePolicy:=false
	for _, line := range strings.Split(found.Data["policy.csv"], "\n") {
		if newPolicy==line {
			duplicatePolicy=true
		}
		log.Info(line)
	 }
	 if duplicatePolicy== false{
		found.Data["policy.csv"]=found.Data["policy.csv"]+"\n"+newPolicy
		err = r.Client.Update(ctx, found)
	}
	//log.Info("The length of the slice is:", len(split))
	// policies := found.Data["policy.csv"]

	// if found.Data["policy.csv"] != "" {
	// 	for _, policy := range found.Data["policy.csv"] {
	// 		policies = append(policies, policy)
			
	// 	}


	//   log.Info(newPolicy)
	//   policies = append(policies, newPolicy)

	// }
// 	rbac := map[string]map[string]string{
// 		"data":{
// 		"policy.csv": "g," +team.Spec.Argo.Tokens.ArgocdUser+"-admin,role: " +req.Name+"-admin",
// 	},
// }
	// rbacByte, _ := json.Marshal(rbac)
	// log.Info(string(rbacByte))

	// err = r.Client.Patch(context.Background(), &corev1.ConfigMap{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Namespace: "argocd",
	// 		Name:      "argocd-rbac-cm",
	// 	},
	// }, client.RawPatch(types.StrategicMergePatchType, rbacByte))
	// if err != nil {
	// 	log.Error(err, "Failed to patch rbac cm")
	// 	return ctrl.Result{}, err
 	// }
	return ctrl.Result{}, nil
}
func (r *TeamReconciler)setRBACArgoCDViewUser(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	reqLogger := logf.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling team")
    team := &teamv1.Team{}
	found := &corev1.ConfigMap{}
	err1 := r.Client.Get(context.TODO(), req.NamespacedName, team)
	if err1 != nil {
		log.Error(err1, "Failed to get  team")
		return ctrl.Result{}, err1
	}
	err := r.Client.Get(ctx, types.NamespacedName{Name: "argocd-rbac-cm", Namespace: "argocd"}, found)
	if err != nil {
		log.Error(err, "Failed to get  cm")
		return ctrl.Result{}, err
	}
	//var policies []string
	// split := strings.Split(found.Data["policy.csv"], "\n")
	// for _, v := range split {
	// 	log.Info((v))
	// }
	log.Info("in setRBACArgoCDUser")
	log.Info(req.Name+"-View-CI")
	newPolicy :="g," +req.Name+"-View-CI,role: " +req.Name+"-view "
    duplicatePolicy:=false
	for _, line := range strings.Split(found.Data["policy.csv"], "\n") {
		if newPolicy==line {
			duplicatePolicy=true
		}
		log.Info(line)
	 }
	 if duplicatePolicy== false{
		found.Data["policy.csv"]=found.Data["policy.csv"]+"\n"+newPolicy
		err = r.Client.Update(ctx, found)
	}
	return ctrl.Result{}, nil
}
// SetupWithManager sets up the controller with the Manager.
func (r *TeamReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&teamv1.Team{}).
		Complete(r)
}
