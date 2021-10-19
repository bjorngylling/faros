Start a local k3s cluster with some exposed ports
```
k3d cluster create -p 30080-30099:30080-30099@server:0
```

Create faros namespace
```
kubectl create namespace faros
```

Register an ingress resource
```
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nginx
  annotations:
    ingress.kubernetes.io/ssl-redirect: "false"
spec:
  rules:
  - http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nginx
            port:
              number: 80
EOF
```

