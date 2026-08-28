# Shading Dashboard

Manage access VLAN assignments on Cisco C3850 switches via NETCONF, with port groups for bulk changes per workstation.

**URL:** https://shading-dashboard.nepsweden.tech

## Stack

- **Backend:** Go, chi, scrapligo (NETCONF), WebSocket
- **Frontend:** Next.js (App Router)

## Local development

**Backend**

```bash
cd backend
cp data/config.example.json data/config.json
go run ./cmd/server
```

**Frontend**

```bash
cd frontend
npm install
npm run dev
```

- Dashboard: http://localhost:3000
- Backend: http://localhost:8080/health
- WebSocket: ws://localhost:8080/ws

### Environment

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Backend HTTP port |
| `CONFIG_PATH` | `data/config.json` | Switch + group config |
| `CORS_ORIGIN` | `*` | CORS allowed origin |
| `BACKEND_URL` | `http://localhost:8080` | Next.js rewrite target |
| `NEXT_PUBLIC_WS_URL` | same-host `/ws` in prod | WebSocket URL for browser |

## Docker Compose

```bash
docker compose up --build
```

Frontend: http://localhost:3001

## Kubernetes

Manifests in `deploy/k8s/`. Requires `ghcr-creds` image pull secret and Traefik.

```bash
kubectl apply -f deploy/k8s/
```

## API

| Method | Path | Description |
|--------|------|-------------|
| WS | `/ws` | Live state + VLAN commands |
| GET/PUT | `/api/config` | Switch credentials + groups |
| PUT | `/api/port-groups` | Save all port groups |
| PUT | `/api/port-groups/{id}/vlan` | Apply VLAN to group (REST) |
| PUT | `/api/switches/{id}/ports/{port}/vlan` | Apply VLAN to port (REST) |
