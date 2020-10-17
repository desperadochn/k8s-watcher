package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"time"
)


type PodGCController struct {
	kubeClient clientset.Interface

	podLister       corelisters.PodLister
	podListerSynced cache.InformerSynced

	deletePod              func(namespace, name string) error
	terminatedPodThreshold int
}
 type OriginalK8s struct {
	config    rest.Config
	clientset *kubernetes.Clientset
	listOption metav1.ListOptions
	namespace string
}

type deployInfo struct {
	Status         string `json:"status"`
	Namespace      string `json:"namespace"`
	Changecause    string `json:"changecause"`
	Changetime     string 	`json:"changetime"`
	Code           int     `json:"code"`
	Label          string   `json:"label"`
}


type InClusterK8s struct {
	config interface{}
	clientset *kubernetes.Clientset
	namespace string
}

func (a *OriginalK8s) Init(Host,CAData, CertData, KeyData string)  {
	a.config.Host = Host
	a.config.CAData, _ = base64.StdEncoding.DecodeString(CAData)
	a.config.CertData, _ = base64.StdEncoding.DecodeString(CertData)
	a.config.KeyData, _ = base64.StdEncoding.DecodeString(KeyData)

}
func (a *OriginalK8s) Auth(Host,Token,CertData string)  {
	a.config.BearerToken = Token
	a.config.Host = Host
	a.config.CertData = []byte(CertData)
	fmt.Println(a.config.Host)
}

func InClusterAuth()  {

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	factory := informers.NewSharedInformerFactory(clientset,0)
	deploymentInformer := factory.Apps().V1().Deployments().Informer()
	podInformer := factory.Core().V1().Pods().Informer()
	stopper := make(chan struct{})
	defer close(stopper)
	defer runtime.HandleCrash()
	factory.Start(stopper)

	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:     DeplyonAdd,
		UpdateFunc: DeployOnUpdate,
		DeleteFunc: DeployOnDelete,

	})
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: PodonAdd,
		UpdateFunc: PodonUpdate,
		DeleteFunc: PodonDelete,
	})

	<-stopper

}

func PodonAdd( Obj interface{})  {
	Pods := Obj.(*corev1.Pod)
	println("add a  pods:",Pods.Name)

}
func PodonUpdate(oldObj, newObj interface{})  {
	oldPods := oldObj.(*corev1.Pod)
	newPods := newObj.(*corev1.Pod)
	println("update a old pods:",oldPods.Name)
	println("update a new pods",newPods.Name)
}
func PodonDelete(obj interface{})  {
	pod := obj.(*corev1.Pod)
	fmt.Println("delete a pod:",pod.Name)
}
func (a *OriginalK8s)RollingUpdateStatus()  {
	clientset, _ := kubernetes.NewForConfig(&a.config)
	factory := informers.NewSharedInformerFactory(clientset,0)
	deploymentInformer := factory.Apps().V1().Deployments().Informer()
	stopper := make(chan struct{})
	defer close(stopper)
	defer runtime.HandleCrash()
	factory.Start(stopper)
	deploymentInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:     DeplyonAdd,
		UpdateFunc: DeployOnUpdate,
		DeleteFunc: DeployOnDelete,
	})
	<-stopper
}

func  DeplyonAdd(obj interface{})  {
	deploy := obj.(*appsv1.Deployment)
	fmt.Println("add a new old deployment", deploy.Name,deploy.Namespace)
}
func DeployOnUpdate(oldObj interface{},newObj interface{}) {
	newDeploy := newObj.(*appsv1.Deployment)
	changetime := time.Now().Format("2006-01-02 15:04:05")
	var gcc *PodGCController
	keyName := newDeploy.Name + newDeploy.Namespace
	println(keyName)
	if newDeploy.Status.UpdatedReplicas == *(newDeploy.Spec.Replicas) &&
		newDeploy.Status.Replicas == *(newDeploy.Spec.Replicas) &&
		newDeploy.Status.AvailableReplicas == *(newDeploy.Spec.Replicas) &&
		newDeploy.Status.ObservedGeneration == newDeploy.Generation {
		fmt.Println("rolling update  success:", newDeploy.Name, newDeploy.Namespace)
		var content string
		content = fmt.Sprintf("%s:%s:%s","发布成功",newDeploy.Name,newDeploy.Namespace)
		SendToDingdingTalk(content)
		appName := newDeploy.Labels["app"]
		for range newDeploy.Annotations {
			changeCause := newDeploy.Annotations["kubernetes.io/change-cause"]
			deployInfo := deployInfo{
				Status:      "success",
				Namespace:   newDeploy.Namespace,
				Changecause: changeCause,
				Changetime: changetime,
				Code: 0,
				Label: appName,
			}
			value, err := json.Marshal(deployInfo)
			if err != nil {
				fmt.Printf("json.Marshal failed, err:%v\n", err)
				return
			}
			DumpToEtcd("10.10.6.2:2379", keyName, string(value))
		}

	} else {
		for range newDeploy.Annotations {
			changeCause := newDeploy.Annotations["kubernetes.io/change-cause"]
			deployInfo := deployInfo{
				Status:      "updateing",
				Namespace:   newDeploy.Namespace,
				Changecause: changeCause,
				Changetime: changetime,
				Code: 1,
			}
			value, err := json.Marshal(deployInfo)
			if err != nil {
				fmt.Printf("json.Marshal failed, err:%v\n", err)
				return
			}
			DumpToEtcd("10.10.6.2:2379", keyName, string(value))
		}
	}
}
func DeployOnDelete(obj interface{})  {
	deploy := obj.(*appsv1.Deployment)
	fmt.Println("delete a deployment",deploy.Name,deploy.Namespace)
}





