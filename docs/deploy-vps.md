# Deploy on a VPS with Docker Compose

This guide shows a simple self-hosted deployment for Go-Ai on a Linux VPS. It uses the existing `Dockerfile`, `docker-compose.yml`, and an optional Caddy reverse proxy for HTTPS.

Fly.io and Render are not required. Any Docker-capable Linux host can run Go-Ai.

## Requirements

- A Linux VPS with SSH access.
- Docker Engine.
- Docker Compose plugin (`docker compose`, not the legacy `docker-compose`).
- A Gemini API key for the default provider.
- Optional OpenRouter API key for fallback or alternative models.
- Optional domain name pointing to the VPS if you want HTTPS with Caddy.

Choose a VPS region where your model providers are reachable and reliable. Provider availability, latency, and network filtering can vary by region.

## 1. Clone the repository

```sh
git clone https://github.com/JakuninOleg/Go-Ai.git
cd Go-Ai
```

## 2. Configure secrets

Copy the example environment file and edit it on the server:

```sh
cp .env.example .env
nano .env
```

Set at least:

```dotenv
PORT=8080
GO_AI_SHARED_SECRET=replace-with-a-long-random-secret
GEMINI_API_KEY=your-gemini-key
GEMINI_BASE_URL=https://generativelanguage.googleapis.com/v1beta/openai
OPENROUTER_API_KEY=
OPENROUTER_BASE_URL=https://openrouter.ai/api/v1
MODEL_REFRESH_INTERVAL=1h
```

Never commit `.env` or paste real provider keys into issues, logs, browser code, or client-side applications. Keep `GO_AI_SHARED_SECRET` server-side and use it only from trusted backends.

## 3. Start Go-Ai

Build and start the container:

```sh
docker compose up -d --build
```

Follow logs:

```sh
docker compose logs -f go-ai
```

Go-Ai writes safe structured logs to stdout. Logs intentionally do not include prompts, request bodies, provider keys, `Authorization` headers, or `.env` values.

## 4. Test the service

From the VPS:

```sh
curl http://localhost:8080/health
```

If port `8080` is exposed through the firewall, test from your machine:

```sh
curl http://SERVER_IP:8080/health
```

Protected API routes require bearer auth:

```sh
curl http://SERVER_IP:8080/v1/models \
  -H "Authorization: Bearer <GO_AI_SHARED_SECRET>"
```

Direct HTTP exposure is useful for a quick smoke test, but HTTPS is strongly recommended for real traffic.

## 5. Add HTTPS with Caddy

The repository includes a host-Caddy example at [`../deploy/caddy/Caddyfile.example`](../deploy/caddy/Caddyfile.example):

```caddyfile
go-ai.example.com {
	reverse_proxy localhost:8080
}
```

Use this variant when Caddy runs directly on the VPS host and Go-Ai publishes `8080` to the host, as in the default `docker-compose.yml`.

Basic flow:

1. Point your domain's `A` or `AAAA` record to the VPS.
2. Install Caddy on the host using the official package for your Linux distribution.
3. Copy the example to Caddy's active config, usually `/etc/caddy/Caddyfile`.
4. Replace `go-ai.example.com` with your real domain.
5. Reload Caddy.
6. Test `https://your-domain.example/health`.

If Caddy runs as another container in the same Docker Compose network, use the commented variant in the example file:

```caddyfile
go-ai.example.com {
	reverse_proxy go-ai:8080
}
```

## Firewall notes

- With host Caddy, allow inbound `80/tcp` and `443/tcp` for HTTP-01/HTTPS traffic.
- Prefer keeping `8080/tcp` private when Caddy is the public entrypoint. You can bind Go-Ai to localhost only by changing the Compose port mapping to `127.0.0.1:8080:8080`.
- If you expose `8080/tcp` directly, protected routes still require `Authorization: Bearer <GO_AI_SHARED_SECRET>`, but HTTPS is still recommended because bearer tokens must not travel over plain HTTP.
- Keep provider dashboards and server SSH access protected separately; Go-Ai's bearer token protects only Go-Ai protected routes.

## Update an existing VPS deployment

Pull the latest code and rebuild the container:

```sh
git pull
docker compose up -d --build
```

Check the logs after updating:

```sh
docker compose logs -f go-ai
```

## Stop or restart

Restart:

```sh
docker compose restart go-ai
```

Stop:

```sh
docker compose down
```

## Troubleshooting

- `curl http://localhost:8080/health` fails: check `docker compose ps` and `docker compose logs -f go-ai`.
- Provider requests fail: verify `.env` values on the server and confirm the VPS region can reach the provider APIs.
- HTTPS certificate is not issued: confirm DNS points to the VPS and ports `80` and `443` are open.
- Chat requests return `401`: verify the backend caller sends `Authorization: Bearer <GO_AI_SHARED_SECRET>` exactly.
