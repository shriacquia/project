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