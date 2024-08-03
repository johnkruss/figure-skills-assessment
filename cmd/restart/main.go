package main

import (
	"context"
	"flag"
	"fmt"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"strings"
	"time"
)

// All ripped from example here: https://github.com/kubernetes/client-go/blob/master/examples/out-of-cluster-client-configuration/main.go
// Just reformatted to better encapsulate functions for readability

func main() {
	kubeClient := setupClient()
	allPods := loadAllPods(kubeClient)
	matchingPods := findPodsContaining(allPods, "database")
	deploysForPods := findDeploymentsAndUpdate(matchingPods, kubeClient)
	restartPods(deploysForPods, matchingPods, kubeClient)
}

//This function assumes all pods are part of a ReplicaSet and owning Deployment
//If it isn't we probably shouldn't be touching it anyway
func findDeploymentsAndUpdate(pods []v1.Pod, client *kubernetes.Clientset) []v12.Deployment {
	var deployments []v12.Deployment
	for _, pod := range pods {
		var replicaSetName string
		//find the name of our ReplicaSet
		for _, owner := range pod.OwnerReferences {
			if owner.Kind == "ReplicaSet" {
				replicaSetName = owner.Name
			}
		}

		//Retrieve the ReplicaSet
		replicaSet, err := client.AppsV1().ReplicaSets("default").Get(context.TODO(), replicaSetName, metaV1.GetOptions{})
		if err != nil {
			panic(err)
		}

		//Use the retrieved ReplicaSet to find the owning Deployment
		for _, owner := range replicaSet.OwnerReferences {
			if owner.Kind == "Deployment" {
				deployment, err := client.AppsV1().Deployments("default").Get(context.TODO(), owner.Name, metaV1.GetOptions{})
				if err != nil {
					panic(err)
				}
				deployments = append(deployments, *deployment)
			}
		}
	}
	return deployments
}

func restartPods(deployments []v12.Deployment, pods []v1.Pod, client *kubernetes.Clientset) {
	//find the specific container definition that needs to be update to force a restart
	for _, deployment := range deployments {
		deploymentUpdated := false
		for i, container := range deployment.Spec.Template.Spec.Containers {
			for _, pod := range pods {
				for _, podContainer := range pod.Spec.Containers {
					fmt.Println("Pod Container Name")
					fmt.Println(podContainer.Name)
					if container.Name == podContainer.Name {
						fmt.Println("Found container to restart")
						deployment.Spec.Template.Spec.Containers[i].Env = append(container.Env, v1.EnvVar{
							Name:  "FORCE_RESTART_TIME",
							Value: time.Now().UTC().String(), // Use a timestamp to ensure it's unique
						})
						deploymentUpdated = true
						break
					}
				}
			}
		}
		if deploymentUpdated {
			fmt.Println("Restarting Deployment")
			fmt.Println(deployment.Name)
			client.AppsV1().Deployments("default").Update(context.TODO(), &deployment, metaV1.UpdateOptions{})
		}
	}
}

func findPodsContaining(pods []v1.Pod, searchString string) []v1.Pod {
	var found []v1.Pod
	for _, pod := range pods {
		if strings.Contains(pod.Name, searchString) {
			found = append(found, pod)
		}
	}
	return found
}

func loadAllPods(client *kubernetes.Clientset) []v1.Pod{
	pods, err := client.CoreV1().Pods("default").List(context.TODO(), metaV1.ListOptions{})

	if err != nil {
		panic(err)
	}
	return pods.Items
}

func setupClient() *kubernetes.Clientset {
	//Assumes existence of config file at ~/.kube/config
	kubeConfig := flag.String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	fmt.Println("Using config for operation")
	fmt.Println(*kubeConfig)

	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)

	if err != nil {
		panic(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return client
}

