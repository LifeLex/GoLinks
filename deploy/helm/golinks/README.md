# GoLinks Helm chart

Deploys [GoLinks](https://github.com/LifeLex/GoLinks) — a single Go binary serving a
golink redirector, a JSON API, and an embedded React SPA, backed by SQLite.

## TL;DR

```bash
# Build & push your own image first (no public registry is published yet):
docker build -t <your-registry>/golinks:0.1.0 .
docker push <your-registry>/golinks:0.1.0

helm install golinks deploy/helm/golinks \
  --namespace golinks --create-namespace \
  --set image.repository=<your-registry>/golinks \
  --set image.tag=0.1.0 \
  --set config.baseUrl=https://golinks.example.com
```

## Prerequisites

- Kubernetes 1.23+
- Helm 3.8+ (Helm 4 also works)
- A default `StorageClass` (or set `persistence.storageClass`) supporting `ReadWriteOnce`
- For TLS: an Ingress controller and [cert-manager](https://cert-manager.io/) with a configured Issuer/ClusterIssuer

## Single-instance by design

GoLinks stores all state in one SQLite file, which only a single process may write.
The chart reflects this and is **not** horizontally scalable:

- `replicas` is hardcoded to **1** (`replicaCount` is ignored except to warn you).
- The Deployment uses the **`Recreate`** strategy so a rollout never runs two pods
  against the same volume / SQLite lock.
- The volume is **`ReadWriteOnce`**.

If you need HA, the app would first need a networked datastore (e.g. Postgres) — out
of scope for this chart.

## Persistence

A single PVC is mounted twice via `subPath`:

| Path        | subPath | Holds                         |
| ----------- | ------- | ----------------------------- |
| `/app/data` | `db`    | the SQLite database           |
| `/app/docs` | `docs`  | uploaded `.md` / `.mdx` files |

Mounting `/app/docs` masks the sample docs baked into the image. With
`persistence.seedDocs=true` (default) an init-container copies those baked docs onto
the volume **only when the docs directory is empty**, so uploads are never clobbered.

Set `persistence.existingClaim` to reuse a PVC you manage yourself. Set
`persistence.enabled=false` for an ephemeral install (data and uploads are lost on
pod restart — fine for evaluation only).

## Security context

The image runs as non-root **UID/GID 1001**. `podSecurityContext.fsGroup: 1001` is
**required** so the process can write the mounted volume; without it the pod
CrashLoops on a permission error. Don't remove it unless you change the image user.

## Ingress + TLS (cert-manager)

```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: golinks.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: golinks-tls
      hosts:
        - golinks.example.com
config:
  baseUrl: https://golinks.example.com   # MUST match the host above
```

`config.baseUrl` builds the redirect target when a keyword is missing, so it must
equal your external scheme + host. The chart prints a warning if they diverge.

## Health checks

All probes hit `GET /healthz` (a 200 JSON endpoint added for orchestration). If you
deploy an older image without that route, set `probes.healthPath: "/"`.

## Key values

| Key                            | Default                              | Notes                                            |
| ------------------------------ | ------------------------------------ | ------------------------------------------------ |
| `image.repository`             | `your-registry.example.com/golinks`  | **Placeholder — must override.**                 |
| `image.tag`                    | `""`                                 | Defaults to chart `appVersion`.                  |
| `config.baseUrl`               | `http://localhost:8080`              | Must match the external URL.                     |
| `config.logLevel`              | `info`                               | `debug` \| `info` \| `warn` \| `error`           |
| `persistence.enabled`          | `true`                               | PVC for DB + docs.                               |
| `persistence.size`             | `1Gi`                                |                                                  |
| `persistence.seedDocs`         | `true`                               | Seed baked-in docs onto the volume on first boot.|
| `persistence.retain`           | `true`                               | Annotate the PVC `helm.sh/resource-policy: keep` so `helm uninstall` keeps the data. |
| `persistence.existingClaim`    | `""`                                 | Reuse a PVC instead of creating one.             |
| `ingress.enabled`              | `false`                              |                                                  |
| `resources`                    | 50m/64Mi → 500m/256Mi                | Conservative; tune to your traffic.              |

See [`values.yaml`](./values.yaml) for the full, annotated list.

## Hardening follow-ups (not enabled by default)

- `securityContext.readOnlyRootFilesystem: true` + an `emptyDir` at `/tmp` (SQLite
  WAL/temp files already live on the data volume).
- An auth layer in front of `POST /api/docs`.
