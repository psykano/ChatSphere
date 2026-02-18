package k8s_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// K8s resource types for YAML parsing

type Metadata struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

type Container struct {
	Name           string          `yaml:"name"`
	Image          string          `yaml:"image"`
	Command        []string        `yaml:"command"`
	Args           []string        `yaml:"args"`
	Ports          []ContainerPort `yaml:"ports"`
	EnvFrom        []EnvFromSource `yaml:"envFrom"`
	Resources      Resources       `yaml:"resources"`
	ReadinessProbe *Probe          `yaml:"readinessProbe"`
	LivenessProbe  *Probe          `yaml:"livenessProbe"`
	VolumeMounts   []VolumeMount   `yaml:"volumeMounts"`
}

type ContainerPort struct {
	ContainerPort int    `yaml:"containerPort"`
	Protocol      string `yaml:"protocol"`
}

type EnvFromSource struct {
	ConfigMapRef *ConfigMapRef `yaml:"configMapRef"`
}

type ConfigMapRef struct {
	Name string `yaml:"name"`
}

type Resources struct {
	Requests ResourceList `yaml:"requests"`
	Limits   ResourceList `yaml:"limits"`
}

type ResourceList struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

type Probe struct {
	HTTPGet             *HTTPGet `yaml:"httpGet"`
	Exec                *Exec   `yaml:"exec"`
	InitialDelaySeconds int     `yaml:"initialDelaySeconds"`
	PeriodSeconds       int     `yaml:"periodSeconds"`
}

type HTTPGet struct {
	Path string `yaml:"path"`
	Port int    `yaml:"port"`
}

type Exec struct {
	Command []string `yaml:"command"`
}

type VolumeMount struct {
	Name      string `yaml:"name"`
	MountPath string `yaml:"mountPath"`
}

type Volume struct {
	Name                  string                 `yaml:"name"`
	PersistentVolumeClaim *PVCVolumeSource       `yaml:"persistentVolumeClaim"`
}

type PVCVolumeSource struct {
	ClaimName string `yaml:"claimName"`
}

type PodSpec struct {
	Containers []Container `yaml:"containers"`
	Volumes    []Volume    `yaml:"volumes"`
}

type PodTemplateSpec struct {
	Metadata Metadata `yaml:"metadata"`
	Spec     PodSpec  `yaml:"spec"`
}

type LabelSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels"`
}

type DeploymentSpec struct {
	Replicas int             `yaml:"replicas"`
	Selector LabelSelector   `yaml:"selector"`
	Template PodTemplateSpec `yaml:"template"`
}

type StatefulSetSpec struct {
	ServiceName string          `yaml:"serviceName"`
	Replicas    int             `yaml:"replicas"`
	Selector    LabelSelector   `yaml:"selector"`
	Template    PodTemplateSpec `yaml:"template"`
}

type ServicePort struct {
	Port       int    `yaml:"port"`
	TargetPort int    `yaml:"targetPort"`
	Protocol   string `yaml:"protocol"`
}

type ServiceSpec struct {
	Selector  map[string]string `yaml:"selector"`
	Ports     []ServicePort     `yaml:"ports"`
	ClusterIP string            `yaml:"clusterIP"`
}

type IngressPath struct {
	Path     string `yaml:"path"`
	PathType string `yaml:"pathType"`
	Backend  struct {
		Service struct {
			Name string `yaml:"name"`
			Port struct {
				Number int `yaml:"number"`
			} `yaml:"port"`
		} `yaml:"service"`
	} `yaml:"backend"`
}

type IngressRule struct {
	Host string `yaml:"host"`
	HTTP struct {
		Paths []IngressPath `yaml:"paths"`
	} `yaml:"http"`
}

type IngressSpec struct {
	IngressClassName string        `yaml:"ingressClassName"`
	Rules            []IngressRule `yaml:"rules"`
}

type PVCSpec struct {
	AccessModes []string `yaml:"accessModes"`
	Resources   struct {
		Requests struct {
			Storage string `yaml:"storage"`
		} `yaml:"requests"`
	} `yaml:"resources"`
}

type K8sResource struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   Metadata    `yaml:"metadata"`
	Data       map[string]string `yaml:"data"`
	Spec       yaml.Node   `yaml:"spec"`
}

func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "..")
}

func k8sDir() string {
	return filepath.Join(projectRoot(), "k8s")
}

func readManifest(t *testing.T, filename string) []K8sResource {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(k8sDir(), filename))
	if err != nil {
		t.Fatalf("failed to read %s: %v", filename, err)
	}
	var resources []K8sResource
	docs := strings.Split(string(data), "---")
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var r K8sResource
		if err := yaml.Unmarshal([]byte(doc), &r); err != nil {
			t.Fatalf("failed to parse %s: %v", filename, err)
		}
		resources = append(resources, r)
	}
	return resources
}

func decodeSpec(t *testing.T, node yaml.Node, target any) {
	t.Helper()
	if err := node.Decode(target); err != nil {
		t.Fatalf("failed to decode spec: %v", err)
	}
}

func TestManifestFilesExist(t *testing.T) {
	expected := []string{
		"namespace.yaml",
		"configmap.yaml",
		"backend.yaml",
		"frontend.yaml",
		"redis.yaml",
		"ingress.yaml",
	}
	for _, f := range expected {
		path := filepath.Join(k8sDir(), f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("missing manifest: k8s/%s", f)
		}
	}
}

func TestNamespace(t *testing.T) {
	resources := readManifest(t, "namespace.yaml")
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	ns := resources[0]
	if ns.Kind != "Namespace" {
		t.Errorf("expected Namespace kind, got %s", ns.Kind)
	}
	if ns.Metadata.Name != "chatsphere" {
		t.Errorf("expected namespace name chatsphere, got %s", ns.Metadata.Name)
	}
}

func TestConfigMap(t *testing.T) {
	resources := readManifest(t, "configmap.yaml")
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	cm := resources[0]
	if cm.Kind != "ConfigMap" {
		t.Errorf("expected ConfigMap kind, got %s", cm.Kind)
	}
	if cm.Metadata.Namespace != "chatsphere" {
		t.Errorf("expected namespace chatsphere, got %s", cm.Metadata.Namespace)
	}
	if cm.Data["LISTEN_ADDR"] != ":8080" {
		t.Errorf("expected LISTEN_ADDR :8080, got %s", cm.Data["LISTEN_ADDR"])
	}
	if cm.Data["REDIS_ADDR"] != "redis:6379" {
		t.Errorf("expected REDIS_ADDR redis:6379, got %s", cm.Data["REDIS_ADDR"])
	}
}

func TestBackendDeployment(t *testing.T) {
	resources := readManifest(t, "backend.yaml")

	var deploy K8sResource
	for _, r := range resources {
		if r.Kind == "Deployment" {
			deploy = r
			break
		}
	}
	if deploy.Kind == "" {
		t.Fatal("backend.yaml should contain a Deployment")
	}
	if deploy.Metadata.Namespace != "chatsphere" {
		t.Errorf("expected namespace chatsphere, got %s", deploy.Metadata.Namespace)
	}

	var spec DeploymentSpec
	decodeSpec(t, deploy.Spec, &spec)

	if spec.Replicas < 2 {
		t.Errorf("backend should have at least 2 replicas, got %d", spec.Replicas)
	}
	if len(spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(spec.Template.Spec.Containers))
	}

	c := spec.Template.Spec.Containers[0]
	if !strings.Contains(c.Image, "backend") {
		t.Errorf("backend container image should contain 'backend', got %s", c.Image)
	}
	if c.Ports[0].ContainerPort != 8080 {
		t.Errorf("expected container port 8080, got %d", c.Ports[0].ContainerPort)
	}
	if c.ReadinessProbe == nil {
		t.Error("backend should have a readiness probe")
	}
	if c.LivenessProbe == nil {
		t.Error("backend should have a liveness probe")
	}
	if c.ReadinessProbe != nil && c.ReadinessProbe.HTTPGet.Path != "/health" {
		t.Errorf("readiness probe should check /health, got %s", c.ReadinessProbe.HTTPGet.Path)
	}
	if c.Resources.Requests.CPU == "" || c.Resources.Requests.Memory == "" {
		t.Error("backend should have resource requests")
	}
	if c.Resources.Limits.CPU == "" || c.Resources.Limits.Memory == "" {
		t.Error("backend should have resource limits")
	}
	if len(c.EnvFrom) == 0 {
		t.Error("backend should reference configmap via envFrom")
	}
}

func TestBackendService(t *testing.T) {
	resources := readManifest(t, "backend.yaml")

	var svc K8sResource
	for _, r := range resources {
		if r.Kind == "Service" {
			svc = r
			break
		}
	}
	if svc.Kind == "" {
		t.Fatal("backend.yaml should contain a Service")
	}

	var spec ServiceSpec
	decodeSpec(t, svc.Spec, &spec)

	if spec.Ports[0].Port != 8080 {
		t.Errorf("backend service port should be 8080, got %d", spec.Ports[0].Port)
	}
	if spec.Selector["app.kubernetes.io/component"] != "backend" {
		t.Error("backend service selector should target backend component")
	}
}

func TestFrontendDeployment(t *testing.T) {
	resources := readManifest(t, "frontend.yaml")

	var deploy K8sResource
	for _, r := range resources {
		if r.Kind == "Deployment" {
			deploy = r
			break
		}
	}
	if deploy.Kind == "" {
		t.Fatal("frontend.yaml should contain a Deployment")
	}
	if deploy.Metadata.Namespace != "chatsphere" {
		t.Errorf("expected namespace chatsphere, got %s", deploy.Metadata.Namespace)
	}

	var spec DeploymentSpec
	decodeSpec(t, deploy.Spec, &spec)

	if spec.Replicas < 2 {
		t.Errorf("frontend should have at least 2 replicas, got %d", spec.Replicas)
	}

	c := spec.Template.Spec.Containers[0]
	if !strings.Contains(c.Image, "frontend") {
		t.Errorf("frontend container image should contain 'frontend', got %s", c.Image)
	}
	if c.Ports[0].ContainerPort != 80 {
		t.Errorf("expected container port 80, got %d", c.Ports[0].ContainerPort)
	}
	if c.ReadinessProbe == nil {
		t.Error("frontend should have a readiness probe")
	}
	if c.LivenessProbe == nil {
		t.Error("frontend should have a liveness probe")
	}
	if c.Resources.Requests.CPU == "" || c.Resources.Requests.Memory == "" {
		t.Error("frontend should have resource requests")
	}
	if c.Resources.Limits.CPU == "" || c.Resources.Limits.Memory == "" {
		t.Error("frontend should have resource limits")
	}
}

func TestFrontendService(t *testing.T) {
	resources := readManifest(t, "frontend.yaml")

	var svc K8sResource
	for _, r := range resources {
		if r.Kind == "Service" {
			svc = r
			break
		}
	}
	if svc.Kind == "" {
		t.Fatal("frontend.yaml should contain a Service")
	}

	var spec ServiceSpec
	decodeSpec(t, svc.Spec, &spec)

	if spec.Ports[0].Port != 80 {
		t.Errorf("frontend service port should be 80, got %d", spec.Ports[0].Port)
	}
}

func TestRedisStatefulSet(t *testing.T) {
	resources := readManifest(t, "redis.yaml")

	var sts K8sResource
	for _, r := range resources {
		if r.Kind == "StatefulSet" {
			sts = r
			break
		}
	}
	if sts.Kind == "" {
		t.Fatal("redis.yaml should contain a StatefulSet")
	}

	var spec StatefulSetSpec
	decodeSpec(t, sts.Spec, &spec)

	if spec.Replicas != 1 {
		t.Errorf("redis should have exactly 1 replica, got %d", spec.Replicas)
	}
	if spec.ServiceName != "redis" {
		t.Errorf("redis statefulset serviceName should be redis, got %s", spec.ServiceName)
	}

	c := spec.Template.Spec.Containers[0]
	if !strings.HasPrefix(c.Image, "redis:") {
		t.Errorf("redis image should start with redis:, got %s", c.Image)
	}
	if c.Ports[0].ContainerPort != 6379 {
		t.Errorf("expected container port 6379, got %d", c.Ports[0].ContainerPort)
	}
	if c.ReadinessProbe == nil {
		t.Error("redis should have a readiness probe")
	}
	if c.LivenessProbe == nil {
		t.Error("redis should have a liveness probe")
	}
	if c.Resources.Requests.Memory == "" {
		t.Error("redis should have memory resource requests")
	}

	hasDataMount := false
	for _, vm := range c.VolumeMounts {
		if vm.MountPath == "/data" {
			hasDataMount = true
		}
	}
	if !hasDataMount {
		t.Error("redis should mount /data volume")
	}
}

func TestRedisService(t *testing.T) {
	resources := readManifest(t, "redis.yaml")

	var svc K8sResource
	for _, r := range resources {
		if r.Kind == "Service" {
			svc = r
			break
		}
	}
	if svc.Kind == "" {
		t.Fatal("redis.yaml should contain a Service")
	}

	var spec ServiceSpec
	decodeSpec(t, svc.Spec, &spec)

	if spec.Ports[0].Port != 6379 {
		t.Errorf("redis service port should be 6379, got %d", spec.Ports[0].Port)
	}
	if spec.ClusterIP != "None" {
		t.Error("redis service should be headless (clusterIP: None)")
	}
}

func TestRedisPVC(t *testing.T) {
	resources := readManifest(t, "redis.yaml")

	var pvc K8sResource
	for _, r := range resources {
		if r.Kind == "PersistentVolumeClaim" {
			pvc = r
			break
		}
	}
	if pvc.Kind == "" {
		t.Fatal("redis.yaml should contain a PersistentVolumeClaim")
	}
	if pvc.Metadata.Namespace != "chatsphere" {
		t.Errorf("expected namespace chatsphere, got %s", pvc.Metadata.Namespace)
	}

	var spec PVCSpec
	decodeSpec(t, pvc.Spec, &spec)

	if len(spec.AccessModes) == 0 || spec.AccessModes[0] != "ReadWriteOnce" {
		t.Error("PVC should have ReadWriteOnce access mode")
	}
	if spec.Resources.Requests.Storage == "" {
		t.Error("PVC should request storage")
	}
}

func TestIngress(t *testing.T) {
	resources := readManifest(t, "ingress.yaml")
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	ing := resources[0]
	if ing.Kind != "Ingress" {
		t.Errorf("expected Ingress kind, got %s", ing.Kind)
	}
	if ing.Metadata.Namespace != "chatsphere" {
		t.Errorf("expected namespace chatsphere, got %s", ing.Metadata.Namespace)
	}

	var spec IngressSpec
	decodeSpec(t, ing.Spec, &spec)

	if len(spec.Rules) == 0 {
		t.Fatal("ingress should have at least one rule")
	}

	paths := spec.Rules[0].HTTP.Paths
	pathMap := make(map[string]IngressPath)
	for _, p := range paths {
		pathMap[p.Path] = p
	}

	if p, ok := pathMap["/api"]; !ok {
		t.Error("ingress should route /api to backend")
	} else if p.Backend.Service.Name != "backend" || p.Backend.Service.Port.Number != 8080 {
		t.Error("/api should route to backend:8080")
	}

	if p, ok := pathMap["/ws"]; !ok {
		t.Error("ingress should route /ws to backend")
	} else if p.Backend.Service.Name != "backend" || p.Backend.Service.Port.Number != 8080 {
		t.Error("/ws should route to backend:8080")
	}

	if p, ok := pathMap["/"]; !ok {
		t.Error("ingress should route / to frontend")
	} else if p.Backend.Service.Name != "frontend" || p.Backend.Service.Port.Number != 80 {
		t.Error("/ should route to frontend:80")
	}
}

func TestIngressWebSocketAnnotations(t *testing.T) {
	resources := readManifest(t, "ingress.yaml")
	ing := resources[0]

	if ing.Metadata.Annotations == nil {
		t.Fatal("ingress should have annotations for WebSocket support")
	}
	if _, ok := ing.Metadata.Annotations["nginx.ingress.kubernetes.io/proxy-read-timeout"]; !ok {
		t.Error("ingress should have proxy-read-timeout annotation for WebSocket support")
	}
}

func TestAllResourcesInNamespace(t *testing.T) {
	files := []string{"configmap.yaml", "backend.yaml", "frontend.yaml", "redis.yaml", "ingress.yaml"}
	for _, f := range files {
		resources := readManifest(t, f)
		for _, r := range resources {
			if r.Metadata.Namespace != "chatsphere" {
				t.Errorf("%s: %s %s should be in chatsphere namespace, got %q",
					f, r.Kind, r.Metadata.Name, r.Metadata.Namespace)
			}
		}
	}
}

func TestAllResourcesHaveLabels(t *testing.T) {
	files := []string{"namespace.yaml", "configmap.yaml", "backend.yaml", "frontend.yaml", "redis.yaml", "ingress.yaml"}
	for _, f := range files {
		resources := readManifest(t, f)
		for _, r := range resources {
			if r.Metadata.Labels == nil {
				t.Errorf("%s: %s %s should have labels", f, r.Kind, r.Metadata.Name)
				continue
			}
			if _, ok := r.Metadata.Labels["app.kubernetes.io/name"]; !ok {
				t.Errorf("%s: %s %s should have app.kubernetes.io/name label",
					f, r.Kind, r.Metadata.Name)
			}
		}
	}
}
