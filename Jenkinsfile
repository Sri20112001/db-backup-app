pipeline {
  agent any

  options {
    timestamps()
    timeout(time: 30, unit: 'MINUTES')
    disableConcurrentBuilds()
    buildDiscarder(logRotator(numToKeepStr: '20'))
  }

  environment {
    COMPOSE_FILE = 'docker-compose.yml'
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Prepare') {
      steps {
        sh '''
          set -e
          # Ensure a `docker compose` implementation exists. Prefer the plugin;
          # otherwise fetch the pinned standalone binary and verify its
          # checksum before executing — never curl|chmod|run blindly.
          if docker compose version >/dev/null 2>&1; then
            echo "docker compose" > .jenkins-compose
          else
            ARCH="$(uname -m)"
            if [ "$ARCH" != "x86_64" ]; then
              echo "no pinned compose checksum for arch $ARCH — add it to Jenkinsfile" >&2
              exit 1
            fi
            EXPECTED="383ce6698cd5d5bbf958d2c8489ed75094e34a77d340404d9f32c4ae9e12baf0"
            mkdir -p .jenkins-bin
            BIN="$WORKSPACE/.jenkins-bin/docker-compose"
            if [ ! -x "$BIN" ]; then
              curl -SL "https://github.com/docker/compose/releases/download/v2.29.7/docker-compose-linux-${ARCH}" \
                -o "$BIN"
              chmod +x "$BIN"
            fi
            ACTUAL="$(sha256sum "$BIN" | cut -d' ' -f1)"
            if [ "$ACTUAL" != "$EXPECTED" ]; then
              echo "compose checksum mismatch: got $ACTUAL" >&2
              rm -f "$BIN"
              exit 1
            fi
            echo "$BIN" > .jenkins-compose
          fi
          echo "compose: $(cat .jenkins-compose)"
          $(cat .jenkins-compose) version
        '''
      }
    }

    stage('Test (server)') {
      steps {
        sh '''
          set -e
          # Ephemeral Postgres so DB-gated tests (migrations, scheduler
          # idempotency, refresh rotation, region seed) actually run instead
          # of skipping. Torn down even when tests fail.
          NET="vg-ci-$BUILD_NUMBER"
          PG="vg-ci-pg-$BUILD_NUMBER"
          # Drop leftovers from an aborted earlier run, then start fresh.
          docker rm -f "$PG" >/dev/null 2>&1 || true
          docker network rm "$NET" >/dev/null 2>&1 || true
          docker network create "$NET" >/dev/null
          docker run -d --name "$PG" --network "$NET" \
            -e POSTGRES_USER=postgres \
            -e POSTGRES_PASSWORD=postgres \
            -e POSTGRES_DB=ci_test \
            postgres:15-alpine >/dev/null
          i=0
          while [ $i -lt 30 ]; do
            if docker exec "$PG" pg_isready -U postgres >/dev/null 2>&1; then break; fi
            i=$((i+1)); sleep 2
          done
          # `|| STATUS=$?` (not bare set -e): a failing suite must still
          # reach the cleanup below instead of leaking pg containers.
          STATUS=0
          docker run --rm --network "$NET" \
            -v "$WORKSPACE/server:/src" -w /src \
            -e DATABASE_URL="host=$PG user=postgres password=postgres dbname=ci_test port=5432 sslmode=disable" \
            golang:1.26-alpine \
            sh -c "go vet ./... && go test ./... && go run golang.org/x/vuln/cmd/govulncheck@latest ./..." || STATUS=$?
          docker rm -f "$PG" >/dev/null 2>&1 || true
          docker network rm "$NET" >/dev/null 2>&1 || true
          exit $STATUS
        '''
      }
    }

    stage('Test (agent)') {
      steps {
        sh '''
          set -e
          docker run --rm \
            -v "$WORKSPACE/agent:/src" -w /src \
            golang:1.26-alpine \
            sh -c "go vet ./... && go test ./... && go run golang.org/x/vuln/cmd/govulncheck@latest ./..."
        '''
      }
    }

    stage('Build (agent windows)') {
      steps {
        sh '''
          set -e
          # Pure-Go cross-compile of the headless agent (service wrapper
          # included) — no C toolchain needed. dist/ is the ONLY source of
          # distributable binaries (git-ignored); tmp/ dev artifacts must
          # never be shipped. The Fyne setup GUI is NOT built here: it needs
          # network module downloads plus platform graphics backends, so GUI
          # packaging requires a Windows build agent
          # (see setup/README.md + docs/WINDOWS_VALIDATION.md).
          #
          # Future Windows-agent pipeline (separate node, does not touch
          # this Linux flow):
          #   windows-agent: checkout -> go mod tidy (setup/) ->
          #     rsrc manifest embed -> go build GUI ->
          #     signtool Authenticode sign + timestamp ->
          #     dist/windows package -> release
          # Existing stages above are untouched.
          docker run --rm \
            -v "$WORKSPACE/agent:/src" -w /src \
            -e GOOS=windows -e GOARCH=amd64 -e CGO_ENABLED=0 \
            golang:1.26-alpine \
            sh -c "go build -trimpath -o dist/windows/VaultGuard-Agent.exe ./cmd/agent && go run ./cmd/verify-artifact --arch amd64 --min-bytes 5000000 dist/windows/VaultGuard-Agent.exe && go build -trimpath -o dist/windows/VaultGuard-Agent-Setup-Console.exe ./cmd/agent-setup && go run ./cmd/verify-artifact --arch amd64 --min-bytes 100000 dist/windows/VaultGuard-Agent-Setup-Console.exe"
        '''
      }
    }
    

    stage('Test (client)') {
      steps {
        sh '''
          set -e
          docker run --rm \
            -v "$WORKSPACE/client:/app" -w /app \
            node:20-alpine \
            sh -c "npm ci --no-audit --no-fund && npm run lint && npm run test && npm audit --audit-level=high"
        '''
      }
    }

    stage('Build') {
      steps {
        sh '''
          set -e
          COMPOSE="$(cat "$WORKSPACE/.jenkins-compose")"
          $COMPOSE build
        '''
      }
    }

    stage('Deploy') {
      environment {
        // Fetch secrets from Jenkins credentials. BACKUP_DB_PASSWORD is
        // required since the compose file gives the agent a least-privilege
        // role instead of the postgres superuser — add this credential
        // (vaultguard-backup-db-password) before the first deploy.
        POSTGRES_PASSWORD = credentials('vaultguard-postgres-password')
        BACKUP_DB_PASSWORD = credentials('vaultguard-backup-db-password')
        JWT_SECRET = credentials('vaultguard-jwt-secret')
        JWT_REFRESH_SECRET = credentials('vaultguard-jwt-refresh-secret')
        ENCRYPTION_KEY = credentials('vaultguard-encryption-key')
      }
      steps {
        sh '''
          set -e
          COMPOSE="$(cat "$WORKSPACE/.jenkins-compose")"

          # Stop old containers
          $COMPOSE down --remove-orphans >/dev/null 2>&1 || true

          # Start up new containers
          # Passing secrets to docker-compose via environment variables
          export POSTGRES_PASSWORD=$POSTGRES_PASSWORD
          export BACKUP_DB_PASSWORD=$BACKUP_DB_PASSWORD
          export JWT_SECRET=$JWT_SECRET
          export JWT_REFRESH_SECRET=$JWT_REFRESH_SECRET
          export ENCRYPTION_KEY=$ENCRYPTION_KEY

          $COMPOSE up -d

          # Clean up dangling images
          docker image prune -f

          # Health gate: migrations run before the API listens, so a 200
          # here means the new version booted AND migrated successfully.
          OK=0
          i=0
          while [ $i -lt 30 ]; do
            if curl -sf http://localhost:7541/vaultguard/api/health >/dev/null 2>&1; then OK=1; break; fi
            i=$((i+1)); sleep 2
          done
          if [ "$OK" != "1" ]; then
            echo "backend unhealthy after deploy — tailing server logs" >&2
            $COMPOSE logs --tail=50 server || true
            exit 1
          fi
          echo "VaultGuard deployed! Frontend on port 7540, Backend on 7541"
        '''
      }
    }
  }
}
