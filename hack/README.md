Start a local kind cluster with some exposed ports
```
kind create cluster --config=kind.yaml
```

Install the Gateway API resources
```
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml
```

Create a sample Gateway setup
```
kubectl apply -f gateway.yaml
```

Deploy faros
```
kind load docker-image faros:latest
kubectl apply -f faros.yaml
```

Create a nginx deployment and service
```
kubectl apply -f nginx.yaml
```

Delete the cluster
```
kind cluster delete
```

