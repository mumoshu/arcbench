# arcbench

`arcbench` is a simple program to automate the process of benchmarking actions-runner-controller.

It is designed to be run against a source repository that contains a set of workflows that are triggered by arcbench, and the Kubernetes cluster that actions-runner-controller is running on.

```
arcbench
|
pushes commits
|
v
source repository
|
triggers workflows
|
v
GitHub Actions
|
notifies the controller
|
v
actions-runner-controller
|
creates and deletes ephemeralrunners and pods
|
v
Kubernetes
```

## Usage

```
Usage: arcbench [options] [command]
  -controller-namespace string
        Namespace where the controller is running (default "arc-systems")
  -output string
        output file
  -runner-namespace string
        Namespace where the runners are created (default "arc-runners")
  -source-repo string
        source repository, e.g., git@github.com:example/repo.git
  -temp-dir string
        temporary directory. If not specified, a temporary directory is created under /path/to/os/temp/arcbench/$timestamp
  -trigger-file string
        trigger file (default "trigger.txt")
  -triggers int
        number of triggers (default 1)
```

```shell
$ go run . -source-repo git@github.com:myorg/myrepo..git -temp-dir /tmp/arcbench/20241101035159 --triggers 10
```

```console
2024/11/01 05:32:46 Pull the latest changes in the source repository git@github.com:myorg/myrepo.git
2024/11/01 05:32:48 Create or update the trigger file trigger.txt
2024/11/01 05:32:50 Create or update the trigger file trigger.txt
2024/11/01 05:32:52 Create or update the trigger file trigger.txt
2024/11/01 05:32:54 Create or update the trigger file trigger.txt
2024/11/01 05:32:56 Create or update the trigger file trigger.txt
2024/11/01 05:32:59 Create or update the trigger file trigger.txt
2024/11/01 05:33:01 Create or update the trigger file trigger.txt
2024/11/01 05:33:03 Create or update the trigger file trigger.txt
2024/11/01 05:33:05 Create or update the trigger file trigger.txt
2024/11/01 05:33:07 Create or update the trigger file trigger.txt
2024/11/01 05:33:10 Observed 6 ephemeral runners and 6 pods
2024/11/01 05:33:10 Still waiting for the completion of the workflow runs...
2024/11/01 05:33:20 Observed 10 ephemeral runners and 10 pods
2024/11/01 05:33:20 Still waiting for the completion of the workflow runs...
2024/11/01 05:33:30 Observed 9 ephemeral runners and 9 pods
2024/11/01 05:33:30 Still waiting for the completion of the workflow runs...
2024/11/01 05:33:40 Observed 6 ephemeral runners and 6 pods
2024/11/01 05:33:40 Still waiting for the completion of the workflow runs...
2024/11/01 05:33:50 Observed 3 ephemeral runners and 3 pods
2024/11/01 05:33:50 Still waiting for the completion of the workflow runs...
2024/11/01 05:34:01 Observed 3 ephemeral runners and 3 pods
2024/11/01 05:34:01 Still waiting for the completion of the workflow runs...
2024/11/01 05:34:11 Observed 3 ephemeral runners and 3 pods
2024/11/01 05:34:11 Still waiting for the completion of the workflow runs...
2024/11/01 05:34:21 Observed 3 ephemeral runners and 3 pods
2024/11/01 05:34:21 Still waiting for the completion of the workflow runs...
2024/11/01 05:34:31 Observed 3 ephemeral runners and 3 pods
2024/11/01 05:34:31 Still waiting for the completion of the workflow runs...
2024/11/01 05:34:41 Elapsed time: 1m55.678372035s
```
