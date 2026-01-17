pipeline {
    agent any

    environment {
        // Set GOPATH if needed, or rely on Go Modules (recommended)
        GO111MODULE = 'on'
    }

    stages {
        stage('Checkout') {
            steps {
                // Checkout the code from the repo that triggered the build
                checkout scm
            }
        }

        stage('Compile') {
            steps {
                // Compile the backend. Output binary named 'bridge-backend'
                sh 'go build -o bridge-backend main.go'
            }
        }

        stage('Deploy') {
            steps {
                // Example: Restarting a service or moving binary
                // NOTE: This depends heavily on your setup (Docker, SSH, Kubernetes)
                echo 'Deploying application...'
                
                // Simple example: Move binary to a run folder and restart service
                sh 'mv bridge-backend /var/www/backend/'
                // sh 'sudo systemctl restart my-go-service'
            }
        }
    }
}