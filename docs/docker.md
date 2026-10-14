# Docker

The API repository owns `.github/workflows/docker.yml`. Pushes to `main` and
manual runs build the API plus `a3318375/media-center-web` into one image:
`ghcr.io/a3318375/media-center:latest`. The manual `web_ref` input selects a
frontend branch, tag or commit. Frontend-only pushes do not trigger this workflow;
run it manually after frontend changes.

For a private frontend repository, configure the API repository Actions secret
`WEB_REPO_TOKEN` with read access to that repository. Public repositories need
no additional token. GHCR publishing uses the workflow's `GITHUB_TOKEN`.

PanSou must be checked out recursively. The Docker build applies the required
embedding lifecycle patches before compiling the API. Node and Go are used only
in build stages; the final container runs one Go process serving both UI and API.

## Local Build

Run from the API directory with the frontend checked out alongside it:

```sh
git submodule update --init --recursive
docker build --build-context web=../media-center-web -t media-center:local .
docker run --rm -p 5175:18080 -v media-center-test-data:/data media-center:local
```

The trial uses port 5175 and a separate volume to avoid the development service
and its database. Open `http://localhost:5175`. An empty volume requires creating
an administrator on first use.

## Deployment

After the first workflow succeeds, run from the directory containing `docker-compose.yml`:

```sh
docker compose pull
docker compose up -d
```

Deployment uses port 5173; stop the existing development frontend first if it
occupies that port. The `./data` mount stores the database, encryption key and
search cache. Preserve the entire data directory when migrating. Do not run two
API instances against the same data directory. Private GHCR images require
`docker login ghcr.io` with package read permission.

The image reads `HTTP_ADDR`, `MEDIA_CENTER_DATA_DIR` and `MEDIA_CENTER_WEB_DIR`.
Development leaves `MEDIA_CENTER_WEB_DIR` unset and continues using Vite.
