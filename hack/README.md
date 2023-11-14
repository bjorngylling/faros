Start a local kind cluster with some exposed ports
```
kind create cluster --config=kind.yaml
```

Install the Gateway API resources
```
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml
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

