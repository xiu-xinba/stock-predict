#!/bin/sh
set -eu

: "${POSTGRES_RUNTIME_USER:?POSTGRES_RUNTIME_USER must be set}"
: "${POSTGRES_RUNTIME_PASSWORD:?POSTGRES_RUNTIME_PASSWORD must be set}"

if [ -n "${POSTGRES_USER:-}" ] && [ "$POSTGRES_RUNTIME_USER" = "$POSTGRES_USER" ]; then
    echo "POSTGRES_RUNTIME_USER must differ from POSTGRES_USER" >&2
    exit 1
fi

if [ -n "${MIGRATION_DATABASE_URL:-}" ]; then
    psql_database="$MIGRATION_DATABASE_URL"
    psql_args="--dbname=$psql_database"
else
    : "${POSTGRES_USER:?POSTGRES_USER must be set}"
    : "${POSTGRES_DB:?POSTGRES_DB must be set}"
    psql_args="--username=$POSTGRES_USER --dbname=$POSTGRES_DB"
fi

# shellcheck disable=SC2086
psql --no-psqlrc --set=ON_ERROR_STOP=1 $psql_args \
    --set=runtime_user="$POSTGRES_RUNTIME_USER" \
    --set=runtime_password="$POSTGRES_RUNTIME_PASSWORD" <<'SQL'
SELECT set_config('stock_predict.runtime_user', :'runtime_user', false);
DO $guard$
BEGIN
    IF current_user = current_setting('stock_predict.runtime_user') THEN
        RAISE EXCEPTION 'runtime role must differ from migration role';
    END IF;
END
$guard$;

SELECT format(
    'CREATE ROLE %I LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT',
    :'runtime_user',
    :'runtime_password'
)
WHERE NOT EXISTS (
    SELECT 1 FROM pg_roles WHERE rolname = :'runtime_user'
)
\gexec

SELECT format(
    'ALTER ROLE %I WITH LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT',
    :'runtime_user',
    :'runtime_password'
)
\gexec

REVOKE CREATE ON SCHEMA public FROM PUBLIC;
SELECT format('REVOKE CREATE ON SCHEMA public FROM %I', :'runtime_user')
\gexec

SELECT format('REVOKE TEMPORARY ON DATABASE %I FROM PUBLIC', current_database())
\gexec
SELECT format(
    'REVOKE TEMPORARY ON DATABASE %I FROM %I',
    current_database(),
    :'runtime_user'
)
\gexec

SELECT format('GRANT CONNECT ON DATABASE %I TO %I', current_database(), :'runtime_user')
\gexec
SELECT format('GRANT USAGE ON SCHEMA public TO %I', :'runtime_user')
\gexec
SELECT format(
    'GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %I',
    :'runtime_user'
)
\gexec
SELECT format(
    'GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO %I',
    :'runtime_user'
)
\gexec

SELECT format(
    'ALTER DEFAULT PRIVILEGES FOR ROLE %I IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %I',
    current_user,
    :'runtime_user'
)
\gexec
SELECT format(
    'ALTER DEFAULT PRIVILEGES FOR ROLE %I IN SCHEMA public GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO %I',
    current_user,
    :'runtime_user'
)
\gexec
SQL
