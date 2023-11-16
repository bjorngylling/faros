Start a local kind cluster and add the Gateway API CRDs
```
kind create cluster --config=kind.yaml
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

Deploy a couple of backend services
```
mage -d sample-app/ dockerBuild && kind load docker-image sample-app:latest
kubectl apply -f backend.yaml
```

Rebuild and deploy faros (in top directory)
```
mage dockerBuild && kind load docker-image faros:latest && kubectl delete pod -l app=faros && stern faros
```

Delete the cluster
```
kind cluster delete
```

