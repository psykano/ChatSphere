package docker_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type ComposeFile struct {
	Services map[string]Service `yaml:"services"`
	Volumes  map[string]any     `yaml:"volumes"`
	Networks map[string]Network `yaml:"networks"`
}

type Network struct {
	Driver string `yaml:"driver"`
}

type Service struct {
	Image       string         `yaml:"image"`
	Build       *Build         `yaml:"build"`
	Ports       []string       `yaml:"ports"`
	Environment []string       `yaml:"environment"`
	DependsOn   map[string]any `yaml:"depends_on"`
	Volumes     []string       `yaml:"volumes"`
	Healthcheck *Healthcheck   `yaml:"healthcheck"`
	Restart     string         `yaml:"restart"`
	Command     string         `yaml:"command"`
	Networks    []string       `yaml:"networks"`
}

type Build struct {
	Context string `yaml:"context"`
}

type Healthcheck struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval"`
	Timeout     string   `yaml:"timeout"`
	Retries     int      `yaml:"retries"`
	StartPeriod string   `yaml:"start_period"`
}

func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	// From backend/internal/docker/ go up 3 levels to project root
	return filepath.Join(filepath.Dir(filename), "..", "..", "..")
}

func readCompose(t *testing.T) ComposeFile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectRoot(), "docker-compose.yml"))
	if err != nil {
		t.Fatalf("failed to read docker-compose.yml: %v", err)
	}
	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		t.Fatalf("failed to parse docker-compose.yml: %v", err)
	}
	return compose
}

func assertPortMapping(t *testing.T, ports []string, expected string) {
	t.Helper()
	for _, p := range ports {
		if p == expected {
			return
		}
	}
	t.Errorf("expected port mapping %s, got %v", expected, ports)
}

func TestDockerComposeHasAllServices(t *testing.T) {
	compose := readCompose(t)

	for _, name := range []string{"backend", "frontend", "redis"} {
		if _, ok := compose.Services[name]; !ok {
			t.Errorf("missing service: %s", name)
		}
	}
	if len(compose.Services) != 3 {
		t.Errorf("expected 3 services, got %d", len(compose.Services))
	}
}

func TestBackendService(t *testing.T) {
	backend := readCompose(t).Services["backend"]

	if backend.Build == nil || backend.Build.Context != "./backend" {
		t.Error("backend build context should be ./backend")
	}
	assertPortMapping(t, backend.Ports, "8080:8080")

	if _, ok := backend.DependsOn["redis"]; !ok {
		t.Error("backend should depend on redis")
	}
	if backend.Healthcheck == nil {
		t.Error("backend should have a healthcheck")
	}

	hasRedisAddr := false
	for _, env := range backend.Environment {
		if strings.Contains(env, "REDIS_ADDR=redis:6379") {
			hasRedisAddr = true
		}
	}
	if !hasRedisAddr {
		t.Error("backend should have REDIS_ADDR=redis:6379 environment variable")
	}
}

func TestFrontendService(t *testing.T) {
	frontend := readCompose(t).Services["frontend"]

	if frontend.Build == nil || frontend.Build.Context != "./frontend" {
		t.Error("frontend build context should be ./frontend")
	}
	assertPortMapping(t, frontend.Ports, "3000:80")

	if _, ok := frontend.DependsOn["backend"]; !ok {
		t.Error("frontend should depend on backend")
	}
}

func TestRedisService(t *testing.T) {
	redis := readCompose(t).Services["redis"]

	if !strings.HasPrefix(redis.Image, "redis:") {
		t.Errorf("redis image should be redis:*, got %s", redis.Image)
	}
	assertPortMapping(t, redis.Ports, "6379:6379")

	if redis.Healthcheck == nil {
		t.Error("redis should have a healthcheck")
	}

	hasDataVolume := false
	for _, v := range redis.Volumes {
		if strings.Contains(v, "redis-data") {
			hasDataVolume = true
		}
	}
	if !hasDataVolume {
		t.Error("redis should mount a persistent data volume")
	}
}

func TestRedisVolumeDefined(t *testing.T) {
	compose := readCompose(t)
	if _, ok := compose.Volumes["redis-data"]; !ok {
		t.Error("redis-data volume should be defined at the top level")
	}
}

func TestDockerfilesExist(t *testing.T) {
	root := projectRoot()
	for _, path := range []string{"backend/Dockerfile", "frontend/Dockerfile"} {
		if _, err := os.Stat(filepath.Join(root, path)); os.IsNotExist(err) {
			t.Errorf("missing Dockerfile: %s", path)
		}
	}
}

func TestBackendDockerfileContent(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(projectRoot(), "backend/Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "FROM golang:") {
		t.Error("should use golang base image")
	}
	if !strings.Contains(content, "AS builder") {
		t.Error("should use multi-stage build")
	}
	if !strings.Contains(content, "EXPOSE 8080") {
		t.Error("should expose port 8080")
	}
}

func TestFrontendDockerfileContent(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(projectRoot(), "frontend/Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "FROM node:") {
		t.Error("should use node base image")
	}
	if !strings.Contains(content, "AS builder") {
		t.Error("should use multi-stage build")
	}
	if !strings.Contains(content, "nginx") {
		t.Error("should use nginx for serving")
	}
}

func TestNginxConfig(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(projectRoot(), "frontend/nginx.conf"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "proxy_pass http://backend:8080") {
		t.Error("should proxy API requests to backend service")
	}
	if !strings.Contains(content, "try_files") {
		t.Error("should have SPA fallback routing")
	}
}

func TestDockerignoreFiles(t *testing.T) {
	root := projectRoot()
	for _, path := range []string{"backend/.dockerignore", "frontend/.dockerignore"} {
		data, err := os.ReadFile(filepath.Join(root, path))
		if os.IsNotExist(err) {
			t.Errorf("missing: %s", path)
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), ".git") {
			t.Errorf("%s should exclude .git", path)
		}
	}

	data, _ := os.ReadFile(filepath.Join(root, "frontend/.dockerignore"))
	if !strings.Contains(string(data), "node_modules") {
		t.Error("frontend .dockerignore should exclude node_modules")
	}
}

func TestNginxWebSocketProxy(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(projectRoot(), "frontend/nginx.conf"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "location /ws") {
		t.Error("should have WebSocket proxy location")
	}
	if !strings.Contains(content, "proxy_http_version 1.1") {
		t.Error("WebSocket proxy should use HTTP/1.1")
	}
	if !strings.Contains(content, `Upgrade $http_upgrade`) {
		t.Error("WebSocket proxy should set Upgrade header")
	}
	if !strings.Contains(content, `Connection "upgrade"`) {
		t.Error("WebSocket proxy should set Connection header")
	}
}

func TestRestartPolicies(t *testing.T) {
	compose := readCompose(t)
	for name, svc := range compose.Services {
		if svc.Restart != "unless-stopped" {
			t.Errorf("service %s should have restart: unless-stopped, got %q", name, svc.Restart)
		}
	}
}

func TestFrontendHealthcheck(t *testing.T) {
	frontend := readCompose(t).Services["frontend"]
	if frontend.Healthcheck == nil {
		t.Error("frontend should have a healthcheck")
	}
}

func TestNetworkDefined(t *testing.T) {
	compose := readCompose(t)
	net, ok := compose.Networks["chatsphere"]
	if !ok {
		t.Fatal("chatsphere network should be defined at the top level")
	}
	if net.Driver != "bridge" {
		t.Errorf("chatsphere network driver should be bridge, got %q", net.Driver)
	}
}

func TestAllServicesOnNetwork(t *testing.T) {
	compose := readCompose(t)
	for name, svc := range compose.Services {
		found := false
		for _, n := range svc.Networks {
			if n == "chatsphere" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("service %s should be on chatsphere network", name)
		}
	}
}

func TestRedisMemoryLimit(t *testing.T) {
	redis := readCompose(t).Services["redis"]
	if !strings.Contains(redis.Command, "--maxmemory") {
		t.Error("redis should have a maxmemory setting for local development")
	}
	if !strings.Contains(redis.Command, "--maxmemory-policy") {
		t.Error("redis should have a maxmemory-policy setting")
	}
}
