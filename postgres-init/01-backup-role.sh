#!/bin/sh
# Creates the least-privilege backup role used by the VaultGuard agent.
# Runs automatically on FIRST database init (fresh volume only). For an
# EXISTING volume, run the same statements manually — see SOP.md
# ("Backup database role"). Idempotent: safe to re-run by hand.
set -e

: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_DB:?POSTGRES_DB is required}"
: "${BACKUP_DB_USER:?BACKUP_DB_USER is required}"
: "${BACKUP_DB_PASSWORD:?BACKUP_DB_PASSWORD is required}"

# NOTE: role name is operator-controlled; quote it, don't interpolate wild input.
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<EOSQL
DO \$\$
BEGIN
	IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '$BACKUP_DB_USER') THEN
		CREATE ROLE "$BACKUP_DB_USER" LOGIN PASSWORD '$BACKUP_DB_PASSWORD';
	ELSE
		ALTER ROLE "$BACKUP_DB_USER" LOGIN PASSWORD '$BACKUP_DB_PASSWORD';
	END IF;
END
\$\$;
GRANT CONNECT ON DATABASE "$POSTGRES_DB" TO "$BACKUP_DB_USER";
GRANT USAGE ON SCHEMA public TO "$BACKUP_DB_USER";
-- Lets pg_dump SELECT every table (present and future) without superuser.
GRANT pg_read_all_data TO "$BACKUP_DB_USER";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO "$BACKUP_DB_USER";
EOSQL
echo "backup role '$BACKUP_DB_USER' ready"
