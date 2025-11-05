# Smart Autoscaler with Live Dashboard

Kubernetes‑native custom autoscaler using CPU/memory/latency signals + real‑time dashboard.
Includes What‑If simulation with k6, Prometheus/Grafana integration, and Helm chart.

## Quick Start (local: Kind/Minikube)
1) **Cluster + Metrics**
```bash
# Minikube
minikube start
kubectl create namespace smart-autoscaler

# Prometheus (simple)
kubectl apply -n smart-autoscaler -f prometheus/prometheus.yaml
kubectl apply -n smart-autoscaler -f grafana/grafana.yaml
kubectl apply -n smart-autoscaler -f grafana/dashboards-configmap.yaml
```

2) **Sample App**
```bash
kubectl apply -n smart-autoscaler -f k8s/sample-app/deployment.yaml
kubectl apply -n smart-autoscaler -f k8s/sample-app/service.yaml
```

3) **Autoscaler (Helm)**
```bash
helm upgrade --install smart-autoscaler helm/smart-autoscaler -n smart-autoscaler
```

4) **Dashboard (React)**
```bash
cd dashboard
npm install
npm run build
# Serve the build in-cluster or locally (dev server):
npm run dev
```
Visit the dashboard, set Prometheus URL (defaults to in-cluster service), and explore live metrics and scaling events.

5) **What‑If Simulation**
```bash
# From your laptop:
k6 run k6/whatif.js
```
This hits the autoscaler `/predict` endpoint with synthetic RPS to preview scale decisions without applying them.

## Repo Structure
- `autoscaler/` – Go controller + HTTP API (`/events`, `/predict`).
- `dashboard/` – React + Vite + Tailwind UI for replicas, metrics, and event timeline.
- `helm/smart-autoscaler/` – Deploy autoscaler + RBAC + Service.
- `k8s/sample-app/` – Demo service exposing CPU/latency metrics.
- `prometheus/` – Single‑file Prometheus config to scrape sample app & autoscaler.
- `grafana/` – Minimal Grafana deployment + dashboard config.
- `k6/` – What‑If script.
- `tests/` – Unit tests for decision engine.

## Notes
- Safe bounds (min/max replicas) and cooldown are configurable via Helm values.
- Event log stored in a ConfigMap and also served by the autoscaler for the dashboard.
- Predictive mode uses a simple moving average placeholder you can replace with ML later.
