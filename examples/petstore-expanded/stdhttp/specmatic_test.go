//go:build specmatic

package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/oapi-codegen/oapi-codegen/v2/examples/petstore-expanded/stdhttp/api"
	"github.com/oapi-codegen/oapi-codegen/v2/examples/petstore-expanded/stdhttp/server"
)

const specmaticImage = "specmatic/specmatic:2.50.1"

func TestSpecmaticContract(t *testing.T) {
	docker, err := exec.LookPath("docker")
	if err != nil {
		t.Fatal("Docker is required to run the Specmatic contract test")
	}

	srv, err := server.NewServer()
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	applicationSpec, err := api.GetSpec()
	if err != nil {
		t.Fatalf("load application OpenAPI document: %v", err)
	}
	applicationHandler := srv.Handler
	testHandler := http.NewServeMux()
	testHandler.Handle("/", applicationHandler)
	testHandler.HandleFunc("/openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(applicationSpec)
	})
	srv.Handler = testHandler

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("failed to listen on dynamic port: %v", err)
	}

	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- srv.Serve(listener)
	}()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("shut down server: %v", err)
		}
		_ = listener.Close()
	})

	port := listener.Addr().(*net.TCPAddr).Port
	localBaseURL := "http://127.0.0.1:" + fmt.Sprint(port)
	waitForServer(t, localBaseURL, serveErrors)
	seedPet(t, localBaseURL, "Spot", "dog", 1000)
	seedPet(t, localBaseURL, "Fido", "cat", 1001)

	runSpecmatic(t, docker, "http://host.docker.internal:"+fmt.Sprint(port))
	select {
	case err := <-serveErrors:
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("server failed: %v", err)
		}
	default:
	}
}

func runSpecmatic(t *testing.T, docker, baseURL string) {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	reportDirectory := filepath.Join(workingDirectory, "build", "reports", "specmatic")
	if err := os.RemoveAll(reportDirectory); err != nil {
		t.Fatalf("remove previous Specmatic report: %v", err)
	}
	cmd := exec.Command(docker,
		"run", "--rm",
		"--add-host", "host.docker.internal:host-gateway",
		"--volume", filepath.Dir(workingDirectory)+":/workspace",
		"--workdir", "/workspace/stdhttp",
		"--env", "PETSTORE_BASE_URL="+baseURL,
		"--env", "PETSTORE_SWAGGER_URL="+baseURL+"/openapi.json",
		"--env", "SPECMATIC_REPORT_DIR=/workspace/stdhttp/build/reports/specmatic",
		specmaticImage,
		"test", "--config", "specmatic.yaml",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Specmatic contract tests failed: %v", err)
	}
}

func waitForServer(t *testing.T, baseURL string, serveErrors <-chan error) {
	t.Helper()
	client := &http.Client{Timeout: 250 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-serveErrors:
			t.Fatalf("server failed before becoming ready: %v", err)
		default:
		}
		response, err := client.Get(baseURL + "/pets")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("server did not become ready at %s", baseURL)
}

func seedPet(t *testing.T, baseURL, name, tag string, expectedID int64) {
	t.Helper()
	body, err := json.Marshal(api.NewPet{Name: name, Tag: &tag})
	if err != nil {
		t.Fatalf("encode seed pet: %v", err)
	}
	response, err := http.Post(baseURL+"/pets", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("seed pet %q: %v", name, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("seed pet %q status = %d; want %d", name, response.StatusCode, http.StatusOK)
	}
	var pet api.Pet
	if err := json.NewDecoder(response.Body).Decode(&pet); err != nil {
		t.Fatalf("decode seeded pet %q: %v", name, err)
	}
	if pet.Id != expectedID {
		t.Fatalf("seeded pet %q ID = %d; want %d", name, pet.Id, expectedID)
	}
}
