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

