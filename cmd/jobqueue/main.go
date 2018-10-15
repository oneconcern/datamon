package main

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// simple k8s client that lists all available pods
// it gets config from $HOME/.kube/config
func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	coreClient := clientset.CoreV1().Pods(apiv1.NamespaceDefault)
	pod, err := coreClient.Create(&apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "echojob-new",
			Labels: map[string]string{
				"com.oneconcern.ingestortype": "echo",
			},
		},
		Spec: apiv1.PodSpec{
			RestartPolicy: apiv1.RestartPolicyOnFailure,
			Containers: []apiv1.Container{
				{
					Name:  "main",
					Image: "alpine",
					Args:  []string{"sh", "-c", "\"echo the job finished $(date)\""},
				},
			},
		},
	})

	//jobsClient := clientset.BatchV1().Jobs(apiv1.NamespaceDefault)
	//job, err := jobsClient.Create(&batchv1.Job{
	//	ObjectMeta: metav1.ObjectMeta{
	//		Name: "echojob",
	//		Labels: map[string]string{
	//			"com.oneconcern.ingestortype": "echo",
	//		},
	//	},
	//	Spec: batchv1.JobSpec{
	//		Template: apiv1.PodTemplateSpec{
	//			ObjectMeta: metav1.ObjectMeta{
	//				Name: "echojob",
	//			},
	//			Spec: apiv1.PodSpec{
	//				RestartPolicy: apiv1.RestartPolicyOnFailure,
	//				Containers: []apiv1.Container{
	//					{
	//						Name:  "main",
	//						Image: "alpine",
	//						Args:  []string{"echo", "the", "job", "finished", "$(date)"},
	//					},
	//				},
	//			},
	//		},
	//	},
	//})

	if err != nil {
		panic(err.Error())
	}
	fmt.Println("job scheduled", pod.Name)
}
