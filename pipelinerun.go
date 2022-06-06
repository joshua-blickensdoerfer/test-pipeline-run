package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	clientbeta "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type tektonClient struct {
	KubeClient        kubernetes.Interface
	PipelineRunClient clientbeta.PipelineRunInterface
}

func main() {

	body := map[string]string{}
	json_body, err := json.Marshal(body)
	if err != nil {
		panic(err.Error())
	}

	adress := "http://" + os.Args[1] + ":8080"

	response, err := http.Post(adress, "application/json", bytes.NewBuffer(json_body))
	if err != nil {
		panic(err.Error())
	}

	var resultJson map[string]interface{}

	json.NewDecoder(response.Body).Decode(&resultJson)

	eventID, ok := resultJson["eventID"]
	if !ok {
		panic("Event ID coult not be retrieved from Eventlistener response")
	}

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	client, err := versioned.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("EventID retrived: %s\n", eventID)

	label := "triggers.tekton.dev/triggers-eventid=" + fmt.Sprint(eventID)
	watcher, err := client.TektonV1beta1().PipelineRuns("default").Watch(context.TODO(), metav1.ListOptions{LabelSelector: label})
	if err != nil {
		log.Fatal(err)
	}

	for event := range watcher.ResultChan() {

		pipelinerun := event.Object.(*v1beta1.PipelineRun)
		for _, condition := range pipelinerun.Status.Conditions {
			fmt.Println(condition.Message)
			switch reason := condition.Reason; reason {
			case "Succeeded":
				fallthrough
			case "Completed":
				fmt.Println("PipelineRun successfully terminated")
				goto successfulPipelineRun
			case "Failed":
				fmt.Println("The PipelineRun failed.")
				panic("PipelineRun Failed")
			case "Cancelled":
				fmt.Println("The PipelineRun was cancelled.")
				panic("PipelineRun Failed")
			case "PipelineRunTimeout":
				fmt.Println("The PipelineRun ran into a timeout.")
				panic("PipelineRun Failed")
			}
		}
	}

successfulPipelineRun:
}
