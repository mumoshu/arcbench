package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("flag", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [command]\n", os.Args[0])
		fs.PrintDefaults()
	}

	var (
		c runner
	)

	fs.StringVar(&c.output, "output", "", "output file")
	fs.StringVar(&c.tempDir, "temp-dir", "", "temporary directory. If not specified, a temporary directory is created under /path/to/os/temp/arcbench/$timestamp")
	fs.StringVar(&c.sourceRepo, "source-repo", "", "source repository, e.g., git@github.com:example/repo.git")
	fs.StringVar(&c.triggerFile, "trigger-file", "trigger.txt", "trigger file")
	fs.StringVar(&c.controllerNamespace, "controller-namespace", "arc-systems", "Namespace where the controller is running")
	fs.StringVar(&c.runnerNamespace, "runner-namespace", "arc-runners", "Namespace where the runners are created")
	fs.IntVar(&c.triggers, "triggers", 1, "number of triggers")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := c.run(); err != nil {
		return err
	}

	return nil
}

// runner is the benchmark runner.
type runner struct {
	output string

	// controllerNamespace is the arc-systems namespace.
	controllerNamespace string

	// runnerNamespace is the arc-runners namespace.
	runnerNamespace string

	// tempDir is the temporary directory to clone the source repository.
	tempDir string

	// sourceRepo is the source repository to clone.
	sourceRepo string

	// triggerFile is the file in the source repository
	// to be created or updated to trigger the workflow.
	// This should be a dedicated file for the benchmark, not create or modified by
	// other programs or users.
	triggerFile string

	// triggers is the number of the times the trigger file is created or updated.
	// Let's say your workflow runs a job per a change in the trigger file,
	// triggers=10 means the workflow job is run 10 times,
	// effectively meaning you are testing how long it takes to run the workflow job 10 times.
	triggers int
}

// run runs the benchmark.
//
// A benchmark is run in the following steps:
//  1. Clone the source repository if it does not exist.
//     Otherwise, pull the latest changes.
//  2. Create or update the trigger file in the source repository.
//     This is done by changing the git worktree, committing the changes,
//     and pushing the changes to the remote repository.
//  3. Repeat the step 2 for the specified number of times.
//     This way we can put expected load on actions-runner-controller.
//  4. Wait for the completion of the workflow runs.
//     We emulate this by waiting for the ephemeral runners to be deleted.
//
// The runner uses:
// - git commands to interact with the source repository.
// - kubectl commands to interact with the Kubernetes cluster.
func (r *runner) run() error {
	var (
		start = time.Now()
	)

	if r.sourceRepo == "" {
		return fmt.Errorf("source repository is required")
	}

	tempDir := r.tempDir
	if tempDir == "" {
		timestamp := time.Now().Format("20060102150405")
		tempDir = filepath.Join(os.TempDir(), "arcbench", timestamp)
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary directory: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, ".git")); err == nil {
		log.Printf("Pull the latest changes in the source repository %s", r.sourceRepo)

		if _, err := r.git("-C", tempDir, "pull"); err != nil {
			return err
		}
	} else {
		log.Printf("Clone the source repository %s to %s", r.sourceRepo, tempDir)

		if _, err := r.git("clone", r.sourceRepo, tempDir); err != nil {
			return err
		}
	}

	for i := 0; i < r.triggers; i++ {
		log.Printf("Create or update the trigger file %s", r.triggerFile)

		if err := r.createOrUpdateTriggerFile(); err != nil {
			return err
		}
	}

	// Wait for the ephemeral runners to be created.
	for {
		ephemeralRunners, err := r.getEphemeralRunners()
		if err != nil {
			return err
		}

		pods, err := r.getEphemeralRunnerPods()
		if err != nil {
			return err
		}

		if len(ephemeralRunners) > 0 || len(pods) > 0 {
			break
		}

		log.Printf("Waiting for the creation of the ephemeral runners...")
	}

	// Wait for the ephemeral runners to be deleted.
	for {
		ephemeralRunners, err := r.getEphemeralRunners()
		if err != nil {
			return err
		}

		pods, err := r.getEphemeralRunnerPods()
		if err != nil {
			return err
		}

		if len(ephemeralRunners) == 0 || len(pods) == 0 {
			break
		}

		log.Printf("Observed %d ephemeral runners and %d pods", len(ephemeralRunners), len(pods))

		log.Printf("Still waiting for the completion of the workflow runs...")

		time.Sleep(10 * time.Second)
	}

	// Complete the benchmark.

	elapsed := time.Since(start)

	log.Printf("Elapsed time: %v", elapsed)

	return nil
}

func (r *runner) createOrUpdateTriggerFile() error {
	content, err := os.ReadFile(filepath.Join(r.tempDir, r.triggerFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read the trigger file: %v", err)
		}

		content = []byte("0")
	}

	n, err := strconv.Atoi(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse the trigger file: %v", err)
	}

	n++

	if err := os.WriteFile(filepath.Join(r.tempDir, r.triggerFile), []byte(strconv.Itoa(n)), 0644); err != nil {
		return fmt.Errorf("failed to write the trigger file: %v", err)
	}

	if _, err := r.git("-C", r.tempDir, "add", r.triggerFile); err != nil {
		return err
	}

	if _, err := r.git("-C", r.tempDir, "commit", "-m", fmt.Sprintf("Update %s", r.triggerFile)); err != nil {
		return err
	}

	if _, err := r.git("-C", r.tempDir, "push"); err != nil {
		return err
	}

	return nil
}

func (r *runner) getEphemeralRunners() ([]ephemeralRunner, error) {
	s, err := r.kubectl("get", "ephemeralrunner", "-n", r.runnerNamespace, "-o", "json")
	if err != nil {
		return nil, err
	}

	var list ephemeralRunnerList
	if err := json.Unmarshal([]byte(s), &list); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the list of ephemeral runners: %v", err)
	}

	return list.Items, nil
}

func (r *runner) getEphemeralRunnerPods() ([]pod, error) {
	s, err := r.kubectl("get", "pod", "-n", r.runnerNamespace, "-o", "json")
	if err != nil {
		return nil, err
	}

	var list podList

	if err := json.Unmarshal([]byte(s), &list); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the list of pods: %v", err)
	}

	return list.Items, nil
}

func (r *runner) kubectl(args ...string) (string, error) {
	return r.runCommand("kubectl", args...)
}

func (r *runner) git(args ...string) (string, error) {
	return r.runCommand("git", args...)
}

func (r *runner) runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %v: %s", name, args[0], err, data)
	}

	return string(data), nil
}

type ephemeralRunner struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
}

type pod struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
}

type ephemeralRunnerList struct {
	Items []ephemeralRunner `json:"items"`
}

type podList struct {
	Items []pod `json:"items"`
}
