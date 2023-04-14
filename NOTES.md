# IDE Operator considerations / kubebuilder notes

1. Namespace constraints
    OOTB kubebuilder scaffolds controller mgr to look cluster-wide. You can restrict it to a namespace or a set of namespaces. If you change the OOTB setting, then the corresponding RBAC should be updated so that the ServiceAccount has access restricted to the set namespaces
    Read:
    1. [Empty main](https://book.kubebuilder.io/cronjob-tutorial/empty-main.html)
    2. [MultiNamespacedCacheBuilder](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder)

2. Designing the API: [Ref](https://book.kubebuilder.io/cronjob-tutorial/api-design.html)
    We need to document the `IdeSpec` assuming the CRD is `Ide` in detail. Use the reference document to ask as many questions as you can, and arrive at the Spec
    1. IDE Containers. 
    This could be encapsulated into a [DeplomentSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#deployment-v1-apps) or a [StatefulSetSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#statefulset-v1-apps)
        1. Container: theia
        2. Container: proxy-app
        3. Container: logging
    2. EnvironmentVariables
    3. ServiceAccount
    4. flexStorage (CRD)

3. Design Status:
    `Ide` statuses need to be thought through. An `Ide` could have the following observed states:
    - Active
    - Hibernated
    ...or should we have `Active`, `Inactive` & `Hibernated` just to keep track of _inactivity_ ?

    3.1 Active
    The `Ide` should keep track of "`Active`" "`Ides`" 
    ```java
    Active []corev1.ObjectReference `json:"active,omitempty"`

    ```

    3.2 Hibernating
    The `Ide Operator` should keep track of "`Hibernating`" "`Ides`" 
    ```java
    Hibernating []corev1.ObjectReference `json:"hibernating,omitempty"`

    ```
    3.3: Do we need any timestamps for the IDE? like `LastActiveTime`, `LastHibernatedTime` and/or `InactiveTime` for recociling to set `Hibernated` state once the `InactiveTime` reaches a certain threshold, say (currently set) `240` minutes.
    If yes, here would be how we could add it:
    ```java
    LastActiveTime *metav1.Time `json:"lastActiveTime,omitempty"`
    LastHibernatedTime *metav1.Time `json:"lastHibernatedTime,omitempty"`
    InactiveTime *metav1.Time `json:"inactiveTime,omitempty"`
    ```

4. Design IDE `IdeState` Machine:

    How do we reconcile (reconcile here is not the `Reconcile` method, its mostly english :P ) the provisioning statuses? While the required status should most logically be `Successful` and a `Reconcile()` method should catch change in status `Events`. But where should we handle this?
    ```java
    - Provisioning
        - In Progress
        - Successful
        - Error
    - Pre-Provisioning
        - In Progress
        - Successful
        - Error
    - De-Provisioning
        - In Progress
        - Successful
        - Error
    ```
    Here, we also need to restrict the allowable transition routes.
    ```mermaid
    stateDiagram-v2
    title: IDE State Machine v1
    [*] --> PRE_PROVISIONING_IN_PROGRESS: Automated based on actual vs desired PP count
    PRE_PROVISIONING_IN_PROGRESS --> PRE_PROVISIONED
    PRE_PROVISIONING_IN_PROGRESS --> PRE_PROVISIONING_ERROR
    PRE_PROVISIONING_ERROR --> DEPROVISIONING_IN_PROGRESS: Automated or Admin instantiated
    DEPROVISIONING_IN_PROGRESS --> DEPROVISIONED
    DEPROVISIONING_IN_PROGRESS --> DEPROVISIONING_ERROR
    PRE_PROVISIONED --> PROVISIONING_IN_PROGRESS: User "Create IDE" Event
    PROVISIONING_IN_PROGRESS --> PROVISIONED: Automated
    PROVISIONING_IN_PROGRESS --> PROVISIONING_ERROR: Automated
    PROVISIONED --> DEPROVISIONING_IN_PROGRESS: User or Admin instantiated
    PROVISIONING_ERROR --> DEPROVISIONING_IN_PROGRESS: Automated or Admin instantiated

    ```
    ---


    ```mermaid
    stateDiagram-v2
    direction TB
    title: IDE State Machine v2
    accTitle: This is accessible title


        classDef notMoving fill:white
        classDef movement font-style:italic;
        classDef badBadEvent fill:#f00,color:white,font-weight:bold,stroke-width:2px,stroke:yellow

        [*] --> PRE_PROVISIONING
        state PRE_PROVISIONING {
            [*] --> PP_IN_PROGRESS
            PP_IN_PROGRESS --> PRE_PROVISIONED
            PP_IN_PROGRESS --> PP_ERROR
        }

        PRE_PROVISIONING --> PROVISIONING
        PRE_PROVISIONING --> DEPROVISIONING

        state PROVISIONING {
            [*] --> PROVISIONING_IN_PROGRESS
            PROVISIONING_IN_PROGRESS --> PROVISIONED
            PROVISIONING_IN_PROGRESS --> PROVISIONING_ERROR
        }

        state DEPROVISIONING {
            [*] --> DP_IN_PROGRESS
            DP_IN_PROGRESS --> DEPROVISIONED
            DP_IN_PROGRESS --> DP_ERROR
        }

    ```
    ---

 5. Implementing the controller:
    We need to list all the possible (english) states that an `Ide` could be in, and understand upfron on what should we be doing about it.




# Miscellaneous:
## Ch 1.9.0: Make Docker build commands
```java
make docker-build IMG=localhost/kubebuilder-cronjob-tutorial:1.9.0-1
```

## Ch 1.9.0: If your `make deploy` fails with this error:
```java
❯ make deploy
test -s /Users/shrinidhijahagirdar/Repos/k8s/project/bin/controller-gen && /Users/shrinidhijahagirdar/Repos/k8s/project/bin/controller-gen --version | grep -q v0.11.1 || \
	GOBIN=/Users/shrinidhijahagirdar/Repos/k8s/project/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.11.1
/Users/shrinidhijahagirdar/Repos/k8s/project/bin/controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
test -s /Users/shrinidhijahagirdar/Repos/k8s/project/bin/kustomize || { curl -Ss "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s -- 3.8.7 /Users/shrinidhijahagirdar/Repos/k8s/project/bin; }
make: *** [/Users/shrinidhijahagirdar/Repos/k8s/project/bin/kustomize] Killed: 9
```
..I've got a solution for you, and better yet..the reason (somewhat).
[#6219](https://github.com/operator-framework/operator-sdk/issues/6219) - Is the reason. `make` checks if `project/bin` director is newer than `project/bin/kustomize` and if it is, it tries to `curl` get the installer and install it via bash script, which somehow doesn't play well. 
How to fix it? Now, its not really a fix, but make the `project/bin/kustomize` newer than bin director. I installed `kustomize` again to overcome.

## Ch 1.9.0: After `make deploy` check the logs of the controller in the `project-system` namespace. You'll find this erro:
```java
❯ k -n project-system logs pod/project-controller-manager-8684c6d4df-s7c6x
2023-04-13T18:56:36Z	INFO	controller-runtime.metrics	Metrics server is starting to listen	{"addr": "127.0.0.1:8080"}
2023-04-13T18:56:36Z	INFO	controller-runtime.builder	Registering a mutating webhook	{"GVK": "batch.tutorial.kubebuilder.io/v1, Kind=CronJob", "path": "/mutate-batch-tutorial-kubebuilder-io-v1-cronjob"}
2023-04-13T18:56:36Z	INFO	controller-runtime.webhook	Registering webhook	{"path": "/mutate-batch-tutorial-kubebuilder-io-v1-cronjob"}
2023-04-13T18:56:36Z	INFO	controller-runtime.builder	Registering a validating webhook	{"GVK": "batch.tutorial.kubebuilder.io/v1, Kind=CronJob", "path": "/validate-batch-tutorial-kubebuilder-io-v1-cronjob"}
2023-04-13T18:56:36Z	INFO	controller-runtime.webhook	Registering webhook	{"path": "/validate-batch-tutorial-kubebuilder-io-v1-cronjob"}
2023-04-13T18:56:36Z	INFO	setup	starting manager
2023-04-13T18:56:36Z	INFO	controller-runtime.webhook.webhooks	Starting webhook server
2023-04-13T18:56:36Z	INFO	Starting server	{"path": "/metrics", "kind": "metrics", "addr": "127.0.0.1:8080"}
2023-04-13T18:56:36Z	INFO	Starting server	{"kind": "health probe", "addr": "[::]:8081"}
2023-04-13T18:56:36Z	INFO	Stopping and waiting for non leader election runnables
2023-04-13T18:56:36Z	INFO	Stopping and waiting for leader election runnables
I0413 18:56:36.105193       1 leaderelection.go:248] attempting to acquire leader lease project-system/80807133.tutorial.kubebuilder.io...
2023-04-13T18:56:36Z	INFO	Starting EventSource	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "source": "kind source: *v1.CronJob"}
2023-04-13T18:56:36Z	INFO	Starting EventSource	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "source": "kind source: *v1.Job"}
2023-04-13T18:56:36Z	INFO	Starting Controller	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob"}
2023-04-13T18:56:36Z	INFO	Starting workers	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "worker count": 1}
2023-04-13T18:56:36Z	INFO	Shutdown signal received, waiting for all workers to finish	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob"}
2023-04-13T18:56:36Z	INFO	All workers finished	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob"}
2023-04-13T18:56:36Z	INFO	Stopping and waiting for caches
2023-04-13T18:56:36Z	INFO	Stopping and waiting for webhooks
2023-04-13T18:56:36Z	INFO	Wait completed, proceeding to shutdown the manager
E0413 18:56:36.106240       1 leaderelection.go:330] error retrieving resource lock project-system/80807133.tutorial.kubebuilder.io: Get "https://10.96.0.1:443/apis/coordination.k8s.io/v1/namespaces/project-system/leases/80807133.tutorial.kubebuilder.io": context canceled
2023-04-13T18:56:36Z	ERROR	setup	problem running manager	{"error": "open /tmp/k8s-webhook-server/serving-certs/tls.crt: no such file or directory"}
main.main
	/workspace/main.go:130
runtime.main
	/usr/local/go/src/runtime/proc.go:250
```

This is because the `main.go:106` checks for `os.getEnv("ENABLE_WEBHOOKS")` which we never passed on to the container. **IF YOU MUST** then a way you could do that is by setting an `ENV` command in the `Dockerfile` with `ENABLE_WEBHOOKS=false` somewhere before the `ENTRYPOINT`. I didn't do it and moved on to the `cert-manager`. The `controller` runs with the `make run` command locally where it gets the `ENABLE_WEBHOOKS=false` hence there is no issue with the container. Moving on..

## Ch 1.9.1: Install `cert-manager`
Key: Don't overthink this! KISS
```java
helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --version v1.11.0 --set installCRDs=true
```

## Ch 1.9.2: Deploying Admission Webhooks

### Deploying `kind` Cluster 
(eventually didn't use Kind cluster, so you may skip this. I was facing issues with installation of cert-manager in the kind cluster due to imagePull issues)
```java
❯ kind create cluster --config=kind-config.yaml --image=kindest/node:v1.25.3
Creating cluster "whook-cluster" ...
 ✓ Ensuring node image (kindest/node:v1.25.3) 🖼
 ✓ Preparing nodes 📦 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
 ✓ Joining worker nodes 🚜
Set kubectl context to "kind-whook-cluster"
You can now use your cluster with:

kubectl cluster-info --context kind-whook-cluster

Have a question, bug, or feature request? Let us know! https://kind.sigs.k8s.io/#community 🙂
❯ k get ns
NAME                 STATUS   AGE
default              Active   28m
kube-node-lease      Active   28m
kube-public          Active   28m
kube-system          Active   28m
local-path-storage   Active   28m
❯ cat kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: whook-cluster
nodes:
- role: control-plane
- role: worker
```
### Building the image again
```java
make docker-build docker-push IMG=shriacquia/kubebuilder-cronjob-tutorial:1.9.2-1
test -s /Users/shrinidhijahagirdar/Repos/k8s/project/bin/controller-gen && /Users/shrinidhijahagirdar/Repos/k8s/project/bin/controller-gen --version | grep -q v0.11.1 || \
	GOBIN=/Users/shrinidhijahagirdar/Repos/k8s/project/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.11.1
/Users/shrinidhijahagirdar/Repos/k8s/project/bin/controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
/Users/shrinidhijahagirdar/Repos/k8s/project/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...
test -s /Users/shrinidhijahagirdar/Repos/k8s/project/bin/setup-envtest || GOBIN=/Users/shrinidhijahagirdar/Repos/k8s/project/bin go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
KUBEBUILDER_ASSETS="/Users/shrinidhijahagirdar/Repos/k8s/project/bin/k8s/1.26.0-darwin-amd64" go test ./... -coverprofile cover.out
?   	tutorial.kubebuilder.io/project	[no test files]
ok  	tutorial.kubebuilder.io/project/api/v1	1.073s	coverage: 1.0% of statements
ok  	tutorial.kubebuilder.io/project/controllers	1.647s	coverage: 0.0% of statements
docker build -t shriacquia/kubebuilder-cronjob-tutorial:1.9.2-1 .
[+] Building 3.8s (18/18) FINISHED
 => [internal] load build definition from Dockerfile                                                     0.0s
 => => transferring dockerfile: 37B                                                                      0.0s
 => [internal] load .dockerignore                                                                        0.0s
 => => transferring context: 35B                                                                         0.0s
 => [internal] load metadata for gcr.io/distroless/static:nonroot                                        1.0s
 => [internal] load metadata for docker.io/library/golang:1.19                                           3.7s
 => [auth] library/golang:pull token for registry-1.docker.io                                            0.0s
 => [builder 1/9] FROM docker.io/library/golang:1.19@sha256:9f2dd04486e84eec72d945b077d568976981d9afed8  0.0s
 => [internal] load build context                                                                        0.0s
 => => transferring context: 4.72kB                                                                      0.0s
 => [stage-1 1/3] FROM gcr.io/distroless/static:nonroot@sha256:149531e38c7e4554d4a6725d7d70593ef9f98813  0.0s
 => CACHED [builder 2/9] WORKDIR /workspace                                                              0.0s
 => CACHED [builder 3/9] COPY go.mod go.mod                                                              0.0s
 => CACHED [builder 4/9] COPY go.sum go.sum                                                              0.0s
 => CACHED [builder 5/9] RUN go mod download                                                             0.0s
 => CACHED [builder 6/9] COPY main.go main.go                                                            0.0s
 => CACHED [builder 7/9] COPY api/ api/                                                                  0.0s
 => CACHED [builder 8/9] COPY controllers/ controllers/                                                  0.0s
 => CACHED [builder 9/9] RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go        0.0s
 => CACHED [stage-1 2/3] COPY --from=builder /workspace/manager .                                        0.0s
 => exporting to image                                                                                   0.0s
 => => exporting layers                                                                                  0.0s
 => => writing image sha256:cdacf04bd55d784be7e2de9cd8b580af358162c9c7590c2687311a9c5c6715fd             0.0s
 => => naming to docker.io/shriacquia/kubebuilder-cronjob-tutorial:1.9.2-1                               0.0s
docker push shriacquia/kubebuilder-cronjob-tutorial:1.9.2-1
The push refers to repository [docker.io/shriacquia/kubebuilder-cronjob-tutorial]
6b1402df2c05: Pushed
4cb10dd2545b: Pushed
d2d7ec0f6756: Pushed
1a73b54f556b: Pushed
e624a5370eca: Pushed
d52f02c6501c: Pushed
ff5700ec5418: Pushed
399826b51fcf: Pushed
6fbdf253bbc2: Pushed
d0157aa0c95a: Pushed
1.9.2-1: digest: sha256:00bc755b8340d988f9cdbd896b96da1e561815b5507f6612eea39f1bebf9932d size: 2402
❯ IMG=shriacquia/kubebuilder-cronjob-tutorial:1.9.2-1
❯ echo $IMG
shriacquia/kubebuilder-cronjob-tutorial:1.9.2-1
```

### Load image into the cluster
```java
❯ IMG=shriacquia/kubebuilder-cronjob-tutorial:1.9.2-1
❯ kind load docker-image $IMG --name whook-cluster
Image: "shriacquia/kubebuilder-cronjob-tutorial:1.9.2-1" with ID "sha256:cdacf04bd55d784be7e2de9cd8b580af358162c9c7590c2687311a9c5c6715fd" not yet present on node "whook-cluster-control-plane", loading...
Image: "shriacquia/kubebuilder-cronjob-tutorial:1.9.2-1" with ID "sha256:cdacf04bd55d784be7e2de9cd8b580af358162c9c7590c2687311a9c5c6715fd" not yet present on node "whook-cluster-worker", loading...
```

### Run the webhook inside the cluster using `make deploy`
Now I am confused. Is the webhook server same as the controller?
Ok..figured. The `project-system` > `deployment.apps/project-controller-manager`'s pods were looking for a `controller:latest` image, which is not correct.
I updated the `Deployment` to change the `manager` pod to have `image: shriacquia/kubebuilder-cronjob-turorial:1.9.2-1`
All set! everything is working!

** Manager Logs **
```java
2023-04-14T13:23:44Z	INFO	controller-runtime.metrics	Metrics server is starting to listen	{"addr": "127.0.0.1:8080"}
2023-04-14T13:23:44Z	INFO	controller-runtime.builder	Registering a mutating webhook	{"GVK": "batch.tutorial.kubebuilder.io/v1, Kind=CronJob", "path": "/mutate-batch-tutorial-kubebuilder-io-v1-cronjob"}
2023-04-14T13:23:44Z	INFO	controller-runtime.webhook	Registering webhook	{"path": "/mutate-batch-tutorial-kubebuilder-io-v1-cronjob"}
2023-04-14T13:23:44Z	INFO	controller-runtime.builder	Registering a validating webhook	{"GVK": "batch.tutorial.kubebuilder.io/v1, Kind=CronJob", "path": "/validate-batch-tutorial-kubebuilder-io-v1-cronjob"}
2023-04-14T13:23:44Z	INFO	controller-runtime.webhook	Registering webhook	{"path": "/validate-batch-tutorial-kubebuilder-io-v1-cronjob"}
2023-04-14T13:23:44Z	INFO	setup	starting manager
2023-04-14T13:23:44Z	INFO	controller-runtime.webhook.webhooks	Starting webhook server
2023-04-14T13:23:44Z	INFO	controller-runtime.certwatcher	Updated current TLS certificate
2023-04-14T13:23:44Z	INFO	Starting server	{"kind": "health probe", "addr": "[::]:8081"}
2023-04-14T13:23:44Z	INFO	Starting server	{"path": "/metrics", "kind": "metrics", "addr": "127.0.0.1:8080"}
2023-04-14T13:23:44Z	INFO	controller-runtime.webhook	Serving webhook server	{"host": "", "port": 9443}
2023-04-14T13:23:44Z	INFO	controller-runtime.certwatcher	Starting certificate watcher
I0414 13:23:44.963369       1 leaderelection.go:248] attempting to acquire leader lease project-system/80807133.tutorial.kubebuilder.io...
I0414 13:23:44.970594       1 leaderelection.go:258] successfully acquired lease project-system/80807133.tutorial.kubebuilder.io
2023-04-14T13:23:44Z	DEBUG	events	project-controller-manager-7bc6f4b84f-bqpwz_3eaf0789-b93a-4c6f-afb1-4a29cd81498f became leader	{"type": "Normal", "object": {"kind":"Lease","namespace":"project-system","name":"80807133.tutorial.kubebuilder.io","uid":"813b0c32-6c58-4ca4-bed1-abceac77569a","apiVersion":"coordination.k8s.io/v1","resourceVersion":"88060"}, "reason": "LeaderElection"}
2023-04-14T13:23:44Z	INFO	Starting EventSource	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "source": "kind source: *v1.CronJob"}
2023-04-14T13:23:44Z	INFO	Starting EventSource	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "source": "kind source: *v1.Job"}
2023-04-14T13:23:44Z	INFO	Starting Controller	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob"}
2023-04-14T13:23:45Z	INFO	Starting workers	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "worker count": 1}
2023-04-14T13:28:31Z	DEBUG	controller-runtime.webhook.webhooks	received request	{"webhook": "/mutate-batch-tutorial-kubebuilder-io-v1-cronjob", "UID": "44935b77-3b09-4baf-9ce8-2dc8f90efb66", "kind": "batch.tutorial.kubebuilder.io/v1, Kind=CronJob", "resource": {"group":"batch.tutorial.kubebuilder.io","version":"v1","resource":"cronjobs"}}
2023-04-14T13:28:31Z	INFO	cronjob-resource	default	{"name": "cronjob-sample"}
2023-04-14T13:28:31Z	DEBUG	controller-runtime.webhook.webhooks	wrote response	{"webhook": "/mutate-batch-tutorial-kubebuilder-io-v1-cronjob", "code": 200, "reason": "", "UID": "44935b77-3b09-4baf-9ce8-2dc8f90efb66", "allowed": true}
2023-04-14T13:28:31Z	DEBUG	controller-runtime.webhook.webhooks	received request	{"webhook": "/validate-batch-tutorial-kubebuilder-io-v1-cronjob", "UID": "ed9e52a1-923f-4eac-9a9f-0065da5b39fc", "kind": "batch.tutorial.kubebuilder.io/v1, Kind=CronJob", "resource": {"group":"batch.tutorial.kubebuilder.io","version":"v1","resource":"cronjobs"}}
2023-04-14T13:28:31Z	INFO	cronjob-resource	validate create	{"name": "cronjob-sample"}
2023-04-14T13:28:31Z	DEBUG	controller-runtime.webhook.webhooks	wrote response	{"webhook": "/validate-batch-tutorial-kubebuilder-io-v1-cronjob", "code": 200, "reason": "", "UID": "ed9e52a1-923f-4eac-9a9f-0065da5b39fc", "allowed": true}
2023-04-14T13:28:31Z	DEBUG	job count	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "CronJob": {"name":"cronjob-sample","namespace":"project-system"}, "namespace": "project-system", "name": "cronjob-sample", "reconcileID": "e501617c-c93c-458c-8cfc-496e7ddcc066", "active jobs": 0, "successful jobs": 0, "failed jobs": 0}
2023-04-14T13:28:31Z	INFO	KubeAPIWarningLogger	unknown field "spec.jobTemplate.metadata.creationTimestamp"
2023-04-14T13:28:31Z	INFO	KubeAPIWarningLogger	unknown field "spec.jobTemplate.spec.template.metadata.creationTimestamp"
2023-04-14T13:28:31Z	DEBUG	no upcoming scheduled times, sleeping until next	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "CronJob": {"name":"cronjob-sample","namespace":"project-system"}, "namespace": "project-system", "name": "cronjob-sample", "reconcileID": "e501617c-c93c-458c-8cfc-496e7ddcc066", "now": "2023-04-14T13:28:31Z", "next run": "2023-04-14T13:29:00Z"}
2023-04-14T13:28:31Z	DEBUG	job count	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "CronJob": {"name":"cronjob-sample","namespace":"project-system"}, "namespace": "project-system", "name": "cronjob-sample", "reconcileID": "6f8ddd35-e66f-454c-989a-2506274971bf", "active jobs": 0, "successful jobs": 0, "failed jobs": 0}
2023-04-14T13:28:31Z	DEBUG	no upcoming scheduled times, sleeping until next	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "CronJob": {"name":"cronjob-sample","namespace":"project-system"}, "namespace": "project-system", "name": "cronjob-sample", "reconcileID": "6f8ddd35-e66f-454c-989a-2506274971bf", "now": "2023-04-14T13:28:31Z", "next run": "2023-04-14T13:29:00Z"}
2023-04-14T13:29:00Z	DEBUG	job count	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "CronJob": {"name":"cronjob-sample","namespace":"project-system"}, "namespace": "project-system", "name": "cronjob-sample", "reconcileID": "7a3a8524-e607-4183-90e8-7bcaa3b4d795", "active jobs": 0, "successful jobs": 0, "failed jobs": 0}
```

### Create CronJob
```java
kubectl create -f config/samples/batch_v1_cronjob.yaml
```

### Check the Manager logs

```java
2023-04-14T13:29:00Z	DEBUG	created Job for CronJob run	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "CronJob": {"name":"cronjob-sample","namespace":"project-system"}, "namespace": "project-system", "name": "cronjob-sample", "reconcileID": "7a3a8524-e607-4183-90e8-7bcaa3b4d795", "now": "2023-04-14T13:29:00Z", "next run": "2023-04-14T13:30:00Z", "current run": "2023-04-14T13:29:00Z", "job": {"namespace": "project-system", "name": "cronjob-sample-1681478940"}}
2023-04-14T13:29:00Z	DEBUG	job count	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "CronJob": {"name":"cronjob-sample","namespace":"project-system"}, "namespace": "project-system", "name": "cronjob-sample", "reconcileID": "4751e7f8-b6ff-4e80-aa44-fd917138063c", "active jobs": 1, "successful jobs": 0, "failed jobs": 0}
2023-04-14T13:29:00Z	DEBUG	no upcoming scheduled times, sleeping until next	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "CronJob": {"name":"cronjob-sample","namespace":"project-system"}, "namespace": "project-system", "name": "cronjob-sample", "reconcileID": "4751e7f8-b6ff-4e80-aa44-fd917138063c", "now": "2023-04-14T13:29:00Z", "next run": "2023-04-14T13:30:00Z"}
2023-04-14T13:29:00Z	DEBUG	job count	{"controller": "cronjob", "controllerGroup": "batch.tutorial.kubebuilder.io", "controllerKind": "CronJob", "CronJob": {"name":"cronjob-sample","namespace":"project-system"}, "namespace": "project-system", "name": "cronjob-sample", "reconcileID": "8ed84d04-8a37-4be4-8ce1-d7ea889933f2", "active jobs": 1, "successful jobs": 0, "failed jobs": 0}
```
...that completes our 1.9.2!
