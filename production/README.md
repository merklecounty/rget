# Production

Kubernetes manifests for the production setup of the `rget server`

## Accessing Services

### rget service

```
kubectl -n sget port-forward sserve-64cfc885c5-pmgpc 8080:2112 --address 0.0.0.0
```

### prometheus

```
kubectl -n sget port-forward prometheus-prometheus-0 9090 --address 0.0.0.0
```
