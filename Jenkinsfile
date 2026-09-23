pipeline {
    agent any

    tools {
        // Assuming Jenkins is configured with these tool names
        go 'Go 1.23'
        nodejs 'NodeJS 20'
    }

    environment {
        // Set Go environments
        GOPATH = "${WORKSPACE}/go"
        GOBIN = "${WORKSPACE}/go/bin"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Build & Test Backend (Server)') {
            steps {
                dir('server') {
                    echo 'Building Server...'
                    sh 'go mod download'
                    sh 'go build -o vaultguard-server cmd/api/main.go'
                    
                    // Run tests if any exist
                    // sh 'go test ./...'
                }
            }
        }

        stage('Build & Test Agent (Worker)') {
            steps {
                dir('agent') {
                    echo 'Building Agent...'
                    sh 'go mod download'
                    sh 'go build -o vaultguard-agent cmd/agent/main.go'
                    
                    // Run tests if any exist
                    // sh 'go test ./...'
                }
            }
        }

        stage('Build Frontend (Client)') {
            steps {
                dir('client') {
                    echo 'Installing Dependencies...'
                    sh 'npm install'
                    
                    echo 'Linting Client...'
                    sh 'npm run lint'
                    
                    echo 'Building Client...'
                    sh 'npm run build'
                }
            }
        }

        stage('Archive Artifacts') {
            steps {
                // Archive the compiled Go binaries
                archiveArtifacts artifacts: 'server/vaultguard-server, agent/vaultguard-agent', fingerprint: true
                
                // Archive the compiled frontend assets (Vite defaults to 'dist')
                archiveArtifacts artifacts: 'client/dist/**/*', fingerprint: true
            }
        }
    }

    post {
        always {
            cleanWs()
        }
        success {
            echo 'VaultGuard CI Pipeline completed successfully!'
        }
        failure {
            echo 'VaultGuard CI Pipeline failed. Please check the logs.'
        }
    }
}
