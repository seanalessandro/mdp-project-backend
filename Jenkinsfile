pipeline {
    agent any

    tools {
        go 'Go For Bridge Backend'
    }

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
                // 1. Move the binary (Works because of Step 1)
                sh 'mv bridge-backend /var/www/backend/'
                
                // 2. Restart the service (Works because of Step 2)
                sh 'sudo systemctl restart my-go-service'
            }
        }
    }
}