# Phase 3 Item 6 - Automated Builds (amd64/arm64)

## Pipelines
- Build multi-arch Docker images via buildx; cache layers; push to registry.
- Separate jobs for api, poller, notifier, web (static), caddy config (if custom image), Android artifacts.

## Caching
- Use build cache from registry; node_modules/go build cache persisted between runs where safe.

## Verification
- Run unit/integration tests before build; scan images for vulnerabilities.
- Fail fast with verbose logs; attach request_id/build_id for banner reference.
