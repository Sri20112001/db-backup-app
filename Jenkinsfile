// VaultGuard CI/CD — Jenkins declarative pipeline.
//
// The agent is expected to have Docker and Go on PATH, and to BE the deploy target
// (same-machine `docker compose up`).
// Works both on a bare-metal agent and from a containerized Jenkins with the
// Docker socket mounted: the pipeline self-provisions a Compose binary if the
// `docker compose` plugin is missing.
//
// Required Jenkins "Secret text" credentials (Manage Jenkins → Credentials):
//   vaultguard-postgres-password
//   vaultguard-jwt-secret          
//   vaultguard-jwt-refresh-secret
//   vaultguard-encryption-key      
//
// Job setup: New Item → Pipeline → "Pipeline script from SCM",
// SCM: Git, Script Path: Jenkinsfile. Trigger via webhook or polling as usual.

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
