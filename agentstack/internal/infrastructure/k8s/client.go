// Package k8s provides Kubernetes client for Knative service management.
/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Knative Service GVR.
var knativeServiceGVR = schema.GroupVersionResource{
	Group:    "serving.knative.dev",
	Version:  "v1",
	Resource: "services",
}

// Client wraps Kubernetes API access for Knative operations.
type Client struct {
	clientset     *kubernetes.Clientset
	dynamicClient dynamic.Interface
	namespace     string
}

// NewClient creates a new Kubernetes client.
func NewClient(kubeconfig, namespace string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		// In-cluster config
		config, err = rest.InClusterConfig()
	} else {
		// Out-of-cluster config
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build k8s config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		clientset:     clientset,
		dynamicClient: dynamicClient,
		namespace:     namespace,
	}, nil
}

// NewMockClient creates a mock client for testing.
func NewMockClient(namespace string) *Client {
	return &Client{
		namespace: namespace,
	}
}

// IsMock returns true if the client is a mock client.
func (c *Client) IsMock() bool {
	return c.clientset == nil
}

// CreateKnativeService creates a Knative Service for an agent.
func (c *Client) CreateKnativeService(ctx context.Context, spec *KnativeServiceSpec) error {
	service := c.buildKnativeService(spec)

	_, err := c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create knative service: %w", err)
	}

	return nil
}

// UpdateKnativeService updates an existing Knative Service.
func (c *Client) UpdateKnativeService(ctx context.Context, spec *KnativeServiceSpec) error {
	existing, err := c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Get(ctx, spec.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing service: %w", err)
	}

	service := c.buildKnativeService(spec)
	service.SetResourceVersion(existing.GetResourceVersion())

	_, err = c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update knative service: %w", err)
	}

	return nil
}

// DeleteKnativeService deletes a Knative Service.
func (c *Client) DeleteKnativeService(ctx context.Context, name string) error {
	err := c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete knative service: %w", err)
	}

	return nil
}

// GetKnativeServiceStatus gets the status of a Knative Service.
func (c *Client) GetKnativeServiceStatus(ctx context.Context, name string) (*ServiceStatus, error) {
	service, err := c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return parseServiceStatus(service)
}

// WatchKnativeService watches for changes to a Knative Service.
func (c *Client) WatchKnativeService(ctx context.Context, name string) (<-chan *ServiceStatus, error) {
	watcher, err := c.dynamicClient.Resource(knativeServiceGVR).
		Namespace(c.namespace).
		Watch(ctx, metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to watch service: %w", err)
	}

	statusCh := make(chan *ServiceStatus, 10)

	go func() {
		defer close(statusCh)
		defer watcher.Stop()

		for event := range watcher.ResultChan() {
			if event.Type == watch.Error {
				continue
			}

			u, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}

			status, err := parseServiceStatus(u)
			if err != nil {
				continue
			}

			select {
			case statusCh <- status:
			case <-ctx.Done():
				return
			}
		}
	}()

	return statusCh, nil
}

// GetNamespace returns the configured namespace.
func (c *Client) GetNamespace() string {
	return c.namespace
}

func (c *Client) buildKnativeService(spec *KnativeServiceSpec) *unstructured.Unstructured {
	container := map[string]interface{}{
		"image": spec.Image,
		"ports": []interface{}{
			map[string]interface{}{
				"containerPort": int64(8080),
				"protocol":      "TCP",
			},
		},
		"env":       buildEnvVars(spec.Env),
		"resources": buildResources(spec.Resources),
	}

	annotations := map[string]interface{}{
		"autoscaling.knative.dev/min-scale":        fmt.Sprintf("%d", spec.MinScale),
		"autoscaling.knative.dev/max-scale":        fmt.Sprintf("%d", spec.MaxScale),
		"autoscaling.knative.dev/target":           fmt.Sprintf("%d", spec.ConcurrencyTarget),
		"autoscaling.knative.dev/scale-down-delay": spec.ScaleDownDelay,
	}

	labels := map[string]interface{}{
		"app.kubernetes.io/name":       spec.Name,
		"app.kubernetes.io/managed-by": "agentstack",
		"agentstack.io/agent-id":       spec.AgentID,
		"agentstack.io/project-id":     spec.ProjectID,
	}

	service := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "serving.knative.dev/v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      spec.Name,
				"namespace": c.namespace,
				"labels":    labels,
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"annotations": annotations,
						"labels":      labels,
					},
					"spec": map[string]interface{}{
						"containers":           []interface{}{container},
						"containerConcurrency": spec.ContainerConcurrency,
						"timeoutSeconds":       spec.TimeoutSeconds,
					},
				},
			},
		},
	}

	if spec.ServiceAccountName != "" {
		if svcSpec, ok := service.Object["spec"].(map[string]interface{}); ok {
			if template, ok := svcSpec["template"].(map[string]interface{}); ok {
				if templateSpec, ok := template["spec"].(map[string]interface{}); ok {
					templateSpec["serviceAccountName"] = spec.ServiceAccountName
				}
			}
		}
	}

	return service
}

func buildEnvVars(env map[string]string) []interface{} {
	vars := make([]interface{}, 0, len(env))
	for k, v := range env {
		vars = append(vars, map[string]interface{}{
			"name":  k,
			"value": v,
		})
	}
	return vars
}

func buildResources(r ResourceSpec) map[string]interface{} {
	return map[string]interface{}{
		"requests": map[string]interface{}{
			"cpu":    r.CPURequest,
			"memory": r.MemoryRequest,
		},
		"limits": map[string]interface{}{
			"cpu":    r.CPULimit,
			"memory": r.MemoryLimit,
		},
	}
}

func parseServiceStatus(u *unstructured.Unstructured) (*ServiceStatus, error) {
	status := &ServiceStatus{
		Conditions: []Condition{},
		Traffic:    []TrafficTarget{},
	}

	statusObj, found, err := unstructured.NestedMap(u.Object, "status")
	if err != nil {
		return nil, err
	}
	if !found {
		return status, nil
	}

	// Parse URL
	if url, ok := statusObj["url"].(string); ok {
		status.URL = url
	}

	// Parse latest revision
	if rev, ok := statusObj["latestReadyRevisionName"].(string); ok {
		status.LatestRevision = rev
	}

	// Parse conditions
	conditions, found, _ := unstructured.NestedSlice(statusObj, "conditions")
	if found {
		for _, c := range conditions {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			condition := Condition{
				Type:   getStringField(cond, "type"),
				Status: getStringField(cond, "status"),
				Reason: getStringField(cond, "reason"),
			}
			status.Conditions = append(status.Conditions, condition)

			if condition.Type == "Ready" && condition.Status == "True" {
				status.Ready = true
			}
		}
	}

	// Parse traffic
	traffic, found, _ := unstructured.NestedSlice(statusObj, "traffic")
	if found {
		for _, t := range traffic {
			tgt, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			target := TrafficTarget{
				Revision: getStringField(tgt, "revisionName"),
				Tag:      getStringField(tgt, "tag"),
			}
			if pct, ok := tgt["percent"].(int64); ok {
				target.Percent = pct
			}
			status.Traffic = append(status.Traffic, target)
		}
	}

	return status, nil
}

func getStringField(m map[string]interface{}, field string) string {
	if v, ok := m[field].(string); ok {
		return v
	}
	return ""
}

// KnativeServiceSpec defines the specification for a Knative Service.
type KnativeServiceSpec struct {
	Name                 string
	AgentID              string
	ProjectID            string
	Image                string
	Env                  map[string]string
	Resources            ResourceSpec
	MinScale             int
	MaxScale             int
	ConcurrencyTarget    int
	ContainerConcurrency int64
	TimeoutSeconds       int64
	ScaleDownDelay       string
	ServiceAccountName   string
}

// ResourceSpec defines resource requests and limits.
type ResourceSpec struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
}

// ServiceStatus represents the status of a Knative Service.
type ServiceStatus struct {
	Ready          bool
	URL            string
	LatestRevision string
	Traffic        []TrafficTarget
	Conditions     []Condition
}

// TrafficTarget represents a traffic routing target.
type TrafficTarget struct {
	Revision string
	Percent  int64
	Tag      string
}

// Condition represents a Kubernetes condition.
type Condition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// WaitForReady waits for a service to become ready.
func (c *Client) WaitForReady(ctx context.Context, name string, timeout time.Duration) (*ServiceStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for service to be ready")
		case <-ticker.C:
			status, err := c.GetKnativeServiceStatus(ctx, name)
			if err != nil {
				continue
			}
			if status.Ready {
				return status, nil
			}
		}
	}
}
