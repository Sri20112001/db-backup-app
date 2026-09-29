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
          # otherwise fetch the standalone binary once into the workspace.
          if docker compose version >/dev/null 2>&1; then
            echo "docker compose" > .jenkins-compose
          else
            mkdir -p .jenkins-bin
            if [ ! -x .jenkins-bin/docker-compose ]; then
              curl -SL "https://github.com/docker/compose/releases/download/v2.29.7/docker-compose-linux-$(uname -m)" \
                -o .jenkins-bin/docker-compose
              chmod +x .jenkins-bin/docker-compose
            fi
            echo "$WORKSPACE/.jenkins-bin/docker-compose" > .jenkins-compose
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
          # go vet + unit tests. DB-backed integration tests self-skip
          # without DATABASE_URL (see server/internal/testutil).
          docker run --rm \
            -v "$WORKSPACE/server:/src" -w /src \
            golang:1.25-alpine \
            sh -c "go vet ./... && go test ./..."
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
            sh -c "npm ci --no-audit --no-fund && npm run lint && npm run test"
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
