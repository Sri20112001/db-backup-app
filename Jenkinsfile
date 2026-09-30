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
          # go vet + unit tests + vulnerability scan. DB-backed integration
          # tests self-skip without DATABASE_URL (see server/internal/testutil).
          docker run --rm \
            -v "$WORKSPACE/server:/src" -w /src \
            golang:1.26-alpine \
            sh -c "go vet ./... && go test ./... && go run golang.org/x/vuln/cmd/govulncheck@latest ./..."
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
        // Fetch secrets from Jenkins credentials
        POSTGRES_PASSWORD = credentials('vaultguard-postgres-password')
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
          export JWT_SECRET=$JWT_SECRET
          export JWT_REFRESH_SECRET=$JWT_REFRESH_SECRET
          export ENCRYPTION_KEY=$ENCRYPTION_KEY

          $COMPOSE up -d
          
          # Clean up dangling images
          docker image prune -f
          
          sleep 5
          echo "VaultGuard deployed! Frontend on port 7540, Backend on 7541"
        '''
      }
    }
  }
}
