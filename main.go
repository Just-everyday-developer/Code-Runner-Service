package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type RunRequest struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

type RunResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
	Passed bool   `json:"passed"`
}

func main() {
	port := "8091"
	http.HandleFunc("/run", handleRun)
	log.Printf("Code runner listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Language != "java" {
		json.NewEncoder(w).Encode(RunResponse{Error: "Unsupported language"})
		return
	}

	output, errStr := runJavaInDocker(req.Code)
	
	json.NewEncoder(w).Encode(RunResponse{
		Output: output,
		Error:  errStr,
		Passed: errStr == "",
	})
}

func runJavaInDocker(code string) (string, string) {
	// Run inside an ephemeral docker container with limits
	// Pass the code via stdin to avoid volume mounting issues (since code-runner is also in a container)
	cmd := exec.Command("docker", "run", "-i", "--rm",
		"--network=none",
		"-m", "128m",
		"--cpus=0.5",
		"-w", "/usr/src/myapp",
		"eclipse-temurin:21-jdk",
		"sh", "-c", "cat > Main.java && javac Main.java && timeout 3 java Main")

	cmd.Stdin = strings.NewReader(code)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Sprintf("Execution Error: %v\nStderr: %s", err, stderr.String())
	}

	return out.String(), ""
}
