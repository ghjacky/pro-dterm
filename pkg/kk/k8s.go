package kk

import (
	"context"
	"dterm/base"

	"github.com/gorilla/websocket"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var Kc *KC

func init() {
	Kc = newK8sClient()
}

type KC struct {
	Client     *kubernetes.Clientset
	RestConfig *rest.Config
	WSClient   *websocket.Dialer
}

func newK8sClient() *KC {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return &KC{
		Client:     client,
		RestConfig: config,
		WSClient:   websocket.DefaultDialer,
	}
}

// func (c *K) ListPods(namespace string) ([]KPod, error) {
// 	ns, err := c.Client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{
// 		TypeMeta: metav1.TypeMeta{
// 			Kind: "Namespace",
// 			APIVersion: "v1",
// 		},
// 	})
// 	if err != nil {

// 	}
// }

func (c *KC) GetPod(namespace, podname string) (*KPod, error) {
	pods := c.Client.CoreV1().Pods(namespace)
	pm, err := pods.Get(context.Background(), podname, metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	})
	if err != nil {
		base.Log.Errorf("failed to get pod (%s): %s", podname, err.Error())
		return nil, err
	}
	return &KPod{
		KC:       c,
		Resource: pods,
		Manifest: *pm,
	}, nil
}
