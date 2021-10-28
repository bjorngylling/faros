Start a local kind cluster with some exposed ports
```
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 8080
    hostPort: 80
    protocol: TCP
EOF
```

Create faros namespace
```
kubectl create namespace faros
```

Create a nginx deployment, service and ingress
```
kubectl apply -f nginx.yaml
```

Deploy faros
```
kind load docker-image faros:latest
kubectl apply -f faros.yaml
```

Delete the cluster
```
kind cluster delete
```

