# Security Policy

## Supported versions

Security fixes are handled on the `main` branch.

## Reporting a vulnerability

Please do not open a public issue that contains API keys, shared secrets, tokens, request bodies with private data, or exploit details.

If you need to report a vulnerability privately, use GitHub's private vulnerability reporting / security advisory flow for this repository if it is available. If not, contact the maintainer through the GitHub profile linked from the repository.

When reporting, include:

- a short description of the issue;
- affected routes or components;
- reproduction steps or proof of concept, with secrets removed;
- the expected impact.

## Secrets handling

- Keep `GO_AI_SHARED_SECRET`, provider API keys, deploy tokens, and `.env` values out of Git.
- Configure runtime secrets in the deployment platform, such as Fly.io or Render.
- Do not log request bodies, provider keys, shared secrets, or `.env` values.
- Use `.env.example` for placeholders only.
