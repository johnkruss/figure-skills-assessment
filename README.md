## Kubernetes Solution

To be able to test my solution and ensure it actually works, I decided to get kubernetes running locally. After a quick
bit of googling it appeared `k3d` would be the simplest tool for the job. A visit to their site lead me to their
installation instructions found [here](https://k3d.io/v5.7.3/#releases) had me setup there:

```curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash```

Their quickstart was sufficient for a tiny test cluster, so we ran:

```k3d cluster create yeehawCluster```

I was then reminded I should go grab kubectl to actually be able to talk to the cluster, so a stop over at
[kubernetes.io](https://kubernetes.io/docs/tasks/tools/install-kubectl-macos/) got me what I needed to get setup there:

```curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/arm64/kubectl"```

From there I dropped the above manifest into a file named `manifest.yaml` and then ran a:

```kubectl apply -f manifest.yaml```

Which successfully exploded, great now let's fix it! The error told me `Deploy` wasn't valid, and a quick ChatGPT
explainer on the `kind` attribute told me I was looking for `Deployment` instead. Great, now we can successfully
apply our manifest.

Next up we run a `kubectl get pods` and see our nginx deploy is in `ImagePullBackOff`

A quick `kubectl describe pod nginx-deploy-88954fc98-ggstw` confirms we're failing to pull `nginx/current` - let's go
find the correct image name. Looks like this is just a simple swap to `latest` instead of `current` as seen out on
[dockerhub](https://hub.docker.com/_/nginx). In a production setting we'd pin to a specific version to avoid our runtime
moving out from under us, but for the sake of this assessment `latest` will be fine. With that change, a subsequent apply,
and a quick check on the pod it appears we're up and running

Ok last bit, let's apply some CPU/Memory restrictions. Another quick trip out to look at
[reference docs](https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/) says we need a resource
section in our manifest - we nest this under `spec.template.spec.containers.resources` and apply again. A subsequent
describe on our pod confirms we've set our requests/limits successfully:

```
Limits:
  cpu:     500m
  memory:  256Mi
Requests:
  cpu:     200m
  memory:  128Mi
```

## Go Solution

Check out `main.go` down in the `cmd/restart` directory

This approach is fairly naive and wouldn't scale without optimization, but it works as follows:
1. Retrieve all pods
2. Search through pods to find any with a name containing our key word
3. Now we backtrack from the pod up through the ReplicaSet to the deployment
4. Now we follow that back down through the deployment template to match containers
5. When we find a match
   1. Update the template with a new environment var
   2. Call it `FORCE_RESTART_TIME`
   3. Make the value a timestamp so collisions are nearly impossible
6. Now update the deployment
   1. The new environment variable will trigger a rolling restart of pods within the updated ReplicaSet

# Welcome 

Welcome to Figure's DevOps skills assessment! 

The goal of this assessment is to get an idea of how you work and your ability to speak in depth about the details in your work. Generally, this assessment should not take you longer than 30 minutes to complete. 

Your answers will be reviewed with you in a subsequent interview.

## Instructions

1. Click on the green "Use This Template" button in the upper-right corner and create a copy of this repository in your own GitHub account.
2. Name your respository and ensure that it's public, as you will need to share it with us for review.
3. When you have completed the questions, please send the URL to the recruiter.

## Assessments

### Kubernetes

1. Fix the issues with this Kubernetes manifest to ensure it is ready for deployment. 
2. Add the following limits and requests to the manifest:
- CPU limit of 0.5 CPU cores
- Memory limit of 256 Mebibytes
- CPU request of 0.2 CPU cores
- Memory request of 128 Mebibytes 

```yaml
apiVersion: apps/v1
kind: Deploy
metadata:
  name: nginx-deploy
  labels:
    app: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:current
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  selector:
    app: nginx
  ports:
    - protocol: TCP
      port: 80
  type: ClusterIP
  ```

### Go

Write a script in Go that redeploys all pods in a Kubernetes cluster that have the word `database` in the name.

Requirements:
- Assume local credentials in your kube config have full access. There is no need to connect via a service account, etc.
- You must use the [client-go](https://github.com/kubernetes/client-go) library.
- Your script must perform a graceful restart, similar to kubectl rollout restart. Do not just delete pods.
- You must use Go modules (no vendor directory).