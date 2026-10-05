# Production Postgres with the least-privilege backup role baked in.
# Replaces the ./postgres-init bind-mount, which cannot work when the
# Docker daemon lives on a different host than the compose client
# (e.g. Jenkins Docker-outside-of-Docker: daemon resolvesbind paths
# against the HOST filesystem, not the Jenkins container).
FROM postgres:15-alpine
COPY postgres-init/01-backup-role.sh /docker-entrypoint-initdb.d/
