# Deployment Manifests

This document covers the Kustomize-based deployment manifests for the Model Registry.

## Overview

The Model Registry uses Kustomize for flexible, layered deployment configurations. The manifest structure follows a base/overlay/options pattern that supports various deployment scenarios.

## Directory Structure

```
manifests/kustomize/
├── base/                          # Core Model Registry
│   ├── model-registry-configmap.yaml
│   ├── model-registry-deployment.yaml
│   ├── model-registry-service.yaml
│   ├── model-registry-sa.yaml
│   └── kustomization.yaml
│
├── overlays/                      # Database configurations
│   ├── db/                        # MySQL-based
│   │   ├── model-registry-db-deployment.yaml
│   │   ├── model-registry-db-pvc.yaml
│   │   ├── model-registry-db-service.yaml
│   │   ├── patches/
│   │   ├── params.env
│   │   ├── secrets.env
│   │   └── kustomization.yaml
│   └── postgres/                  # PostgreSQL-based
│       └── (similar structure)
│
└── options/                       # Optional components
    ├── csi/                       # KServe CSI integration
    ├── catalog/                   # Model Catalog service
    ├── controller/                # InferenceService controller
    ├── istio/                     # Istio service mesh
    └── ui/                        # Model Registry UI
```

## Base Manifests

### Core Components

**Location**: `manifests/kustomize/base/`

#### Deployment

```yaml
# model-registry-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: model-registry-deployment
  labels:
    component: model-registry
spec:
  replicas: 1
  selector:
    matchLabels:
      component: model-registry
  template:
    spec:
      serviceAccountName: model-registry-server
      containers:
      - name: rest-container
        image: ghcr.io/kubeflow/model-registry/server:latest
        command:
        - /model-registry
        - proxy
        - --hostname=0.0.0.0
        - --port=8080
        - --datastore-type=embedmd
        ports:
        - containerPort: 8080
          name: http-api
        livenessProbe:
          httpGet:
            path: /readyz/isDirty
            port: http-api
        readinessProbe:
          httpGet:
            path: /readyz/health
            port: http-api
        startupProbe:
          httpGet:
            path: /readyz/isDirty
            port: http-api
```

#### Service

```yaml
# model-registry-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: model-registry-service
  labels:
    component: model-registry
  annotations:
    kubeflow-component-name: model-registry
spec:
  type: ClusterIP
  ports:
  - port: 8080
    protocol: TCP
    name: http-api
  selector:
    component: model-registry
```

#### ConfigMap

```yaml
# model-registry-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: model-registry-config
data:
  MR_REST_SERVICE_HOST: model-registry-service
  MR_REST_SERVICE_PORT: "8080"
  MR_DATASTORE_TYPE: embedmd
```

## Database Overlays

### MySQL Overlay

**Location**: `manifests/kustomize/overlays/db/`

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: kubeflow

resources:
- ../../base
- model-registry-db-deployment.yaml
- model-registry-db-pvc.yaml
- model-registry-db-service.yaml

configMapGenerator:
- name: model-registry-db-params
  envs:
  - params.env
  options:
    disableNameSuffixHash: true

secretGenerator:
- name: model-registry-db-secrets
  envs:
  - secrets.env
  options:
    disableNameSuffixHash: true

patches:
- path: patches/db-connection.yaml
  target:
    kind: Deployment
    name: model-registry-deployment

images:
- name: mysql
  newName: mysql
  newTag: "8.3"
```

#### Database Deployment

```yaml
# model-registry-db-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: model-registry-db
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: mysql
        image: mysql:8.3
        env:
        - name: MYSQL_DATABASE
          value: metadb
        - name: MYSQL_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: model-registry-db-secrets
              key: MYSQL_ROOT_PASSWORD
        ports:
        - containerPort: 3306
        volumeMounts:
        - name: mysql-storage
          mountPath: /var/lib/mysql
        readinessProbe:
          exec:
            command:
            - mysql
            - -h
            - localhost
            - -uroot
            - -p$(MYSQL_ROOT_PASSWORD)
            - -e
            - "SELECT 1"
      volumes:
      - name: mysql-storage
        persistentVolumeClaim:
          claimName: metadata-mysql
```

### PostgreSQL Overlay

**Location**: `manifests/kustomize/overlays/postgres/`

Similar structure to MySQL, using:
- Image: `postgres:16-alpine`
- Port: 5432
- Environment: `POSTGRES_*` variables

## Optional Components

### CSI Option

**Location**: `manifests/kustomize/options/csi/`

Registers `model-registry://` protocol with KServe:

```yaml
# clusterstoragecontainer.yaml
apiVersion: serving.kserve.io/v1alpha1
kind: ClusterStorageContainer
metadata:
  name: model-registry-storage-initializer
spec:
  container:
    name: storage-initializer
    image: ghcr.io/kubeflow/model-registry/storage-initializer:latest
    env:
    - name: MODEL_REGISTRY_BASE_URL
      value: "model-registry-service.kubeflow.svc.cluster.local:8080"
  supportedUriFormats:
  - prefix: model-registry://
```

### Catalog Option

**Location**: `manifests/kustomize/options/catalog/`

```
catalog/
├── base/                         # Core catalog service
│   ├── catalog-deployment.yaml
│   ├── catalog-service.yaml
│   ├── catalog-postgres-statefulset.yaml
│   └── kustomization.yaml
├── options/
│   └── istio/                    # Istio integration
└── overlays/
    ├── demo/                     # Pre-populated test data
    │   ├── dev-catalog-sources.yaml
    │   ├── dev-community-catalog.yaml
    │   ├── dev-mcp-catalog-sources.yaml
    │   └── dev-community-mcp-servers.yaml
    └── odh/                      # Open Data Hub config
```

### Controller Option

**Location**: `manifests/kustomize/options/controller/`

```
controller/
├── default/
│   └── kustomization.yaml
├── manager/
│   ├── manager.yaml              # Controller deployment
│   └── kustomization.yaml
├── rbac/
│   ├── role.yaml                 # ClusterRole
│   ├── role_binding.yaml
│   ├── service_account.yaml
│   └── kustomization.yaml
├── metrics/
│   └── metrics_service.yaml      # Prometheus metrics
├── network-policy/
│   └── allow-metrics-traffic.yaml
└── overlays/
    └── base/
        └── kustomization.yaml
```

### Istio Option

**Location**: `manifests/kustomize/options/istio/`

```yaml
# virtual-service.yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: model-registry-vs
spec:
  hosts:
  - "*"
  gateways:
  - kubeflow/kubeflow-gateway
  http:
  - match:
    - uri:
        prefix: /api/model_registry/
    route:
    - destination:
        host: model-registry-service
        port:
          number: 8080
```

```yaml
# destination-rule.yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: model-registry-dr
spec:
  host: model-registry-service
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
```

### UI Option

**Location**: `manifests/kustomize/options/ui/`

```
ui/
├── base/
│   ├── ui-deployment.yaml
│   ├── ui-service.yaml
│   └── kustomization.yaml
└── overlays/
    ├── istio/                    # Istio integration
    ├── kubeflow/                 # Kubeflow dashboard
    └── standalone/               # Standalone with auth proxy
```

## Deployment Patterns

### Pattern 1: Kubeflow Central Dashboard

```bash
# Deploy with MySQL database
kubectl apply -k manifests/kustomize/overlays/db -n kubeflow

# Add Istio integration
kubectl apply -k manifests/kustomize/options/istio -n kubeflow

# Optional: Add UI
kubectl apply -k manifests/kustomize/options/ui/overlays/kubeflow -n kubeflow
```

### Pattern 2: Standalone Installation

```bash
# Create namespace
kubectl create namespace model-registry

# Deploy with PostgreSQL
kubectl apply -k manifests/kustomize/overlays/postgres -n model-registry

# Add UI with auth proxy
kubectl apply -k manifests/kustomize/options/ui/overlays/standalone -n model-registry
```

### Pattern 3: Full Stack with Catalog

```bash
# Core Model Registry
kubectl apply -k manifests/kustomize/overlays/db -n kubeflow

# Model Catalog with demo data
kubectl apply -k manifests/kustomize/options/catalog/overlays/demo -n kubeflow

# Istio integration
kubectl apply -k manifests/kustomize/options/istio -n kubeflow
kubectl apply -k manifests/kustomize/options/catalog/options/istio -n kubeflow
```

### Pattern 4: With KServe CSI Driver

```bash
# Deploy CSI (cluster-scoped)
kubectl apply -k manifests/kustomize/options/csi

# Deploy Model Registry
kubectl apply -k manifests/kustomize/overlays/db -n kubeflow
```

### Pattern 5: With Controller

```bash
# Deploy Model Registry
kubectl apply -k manifests/kustomize/overlays/db -n kubeflow

# Deploy InferenceService controller
kubectl apply -k manifests/kustomize/options/controller/overlays/base -n kubeflow
```

## Configuration Management

### ConfigMaps

```yaml
# params.env
MYSQL_DATABASE=metadb
MR_REST_SERVICE_HOST=model-registry-service
MR_REST_SERVICE_PORT=8080
```

### Secrets

```yaml
# secrets.env (not committed)
MYSQL_ROOT_PASSWORD=<password>
MYSQL_USER_NAME=<username>
MYSQL_USER_PASSWORD=<password>
```

### Image Overrides

```yaml
# In kustomization.yaml
images:
- name: ghcr.io/kubeflow/model-registry/server
  newTag: v0.2.0
- name: mysql
  newTag: "8.3"
```

### Patches

```yaml
# patches/db-connection.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: model-registry-deployment
spec:
  template:
    spec:
      containers:
      - name: rest-container
        env:
        - name: MR_DATABASE_HOST
          value: model-registry-db
        - name: MR_DATABASE_PORT
          value: "3306"
        - name: MR_DATABASE_NAME
          valueFrom:
            configMapKeyRef:
              name: model-registry-db-params
              key: MYSQL_DATABASE
```

### Replacements

```yaml
# Dynamic value substitution
replacements:
- source:
    kind: Service
    name: model-registry-db
    fieldPath: spec.ports[0].port
  targets:
  - select:
      kind: ConfigMap
      name: model-registry-config
    fieldPaths:
    - data.MR_DATABASE_PORT
```

## Security Considerations

### Pod Security

```yaml
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: rest-container
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop: ["ALL"]
          readOnlyRootFilesystem: true
```

### RBAC

Controller and catalog components include:
- ServiceAccounts
- Roles/ClusterRoles
- RoleBindings/ClusterRoleBindings

### Network Policies

```yaml
# network-policy/allow-metrics-traffic.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-metrics-traffic
spec:
  podSelector:
    matchLabels:
      component: controller
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: monitoring
    ports:
    - protocol: TCP
      port: 8443
```

## Troubleshooting

### Database Migration Errors

```
error connecting to datastore: Dirty database version {version}
```

**Solution:**
```bash
kubectl exec <db-pod> -- mysql -h localhost -D metadb -u root -p"$password" \
  -e "UPDATE schema_migrations SET dirty = 0;"
```

### Verify Deployment

```bash
# Check pods
kubectl get pods -l component=model-registry -n kubeflow

# Check service
kubectl get svc model-registry-service -n kubeflow

# Check logs
kubectl logs -l component=model-registry -n kubeflow

# Test API
kubectl port-forward svc/model-registry-service 8080:8080 -n kubeflow
curl http://localhost:8080/api/model_registry/v1alpha3/registered_models
```

---

[Back to Kubernetes Index](./README.md) | [Previous: CSI Driver](./csi-driver.md)
