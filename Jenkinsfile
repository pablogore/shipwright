// Jenkinsfile for Syntegrity Dagger
// This is an example configuration that can be customized per project

pipeline {
    agent any
    
    environment {
        GO_VERSION = '1.25.5'
        CACHE_VERSION = 'v1'
        DAGGER_VERSION = 'latest'
    }
    
    options {
        timeout(time: 1, unit: 'HOURS')
        timestamps()
        ansiColor('xterm')
    }
    
    stages {
        stage('Setup') {
            steps {
                echo '🔧 Setting up Go environment...'
                sh '''
                    go env -w GOPRIVATE=github.com/getsyntegrity/*,gitlab.com/syntegrity/*
                    go env -w GONOPROXY=github.com/getsyntegrity/*,gitlab.com/syntegrity/*
                    go env -w GONOSUMDB=github.com/getsyntegrity/*,gitlab.com/syntegrity/*
                    go mod download
                    go mod verify
                '''
            }
        }
        
        stage('Build') {
            steps {
                echo '🔨 Building application...'
                sh 'go build ./...'
            }
        }
        
        stage('Test') {
            parallel {
                stage('Unit Tests') {
                    steps {
                        echo '🧪 Running unit tests...'
                        sh '''
                            go test -race -coverprofile=coverage.out ./...
                            go tool cover -func=coverage.out
                        '''
                    }
                    post {
                        always {
                            publishCoverage adapters: [goCoverageAdapter('coverage.out')]
                        }
                    }
                }
                
                stage('Lint') {
                    steps {
                        echo '🔍 Running linter...'
                        sh 'golangci-lint run --timeout 5m ./...'
                    }
                }
            }
        }
        
        stage('Security') {
            when {
                anyOf {
                    branch 'main'
                    branch 'develop'
                }
            }
            steps {
                echo '🔒 Running security scan...'
                sh '''
                    go install golang.org/x/vuln/cmd/govulncheck@latest
                    govulncheck ./... || exit 1
                '''
            }
        }
        
        stage('Package') {
            when {
                anyOf {
                    branch 'main'
                    branch 'develop'
                }
            }
            steps {
                echo '📦 Packaging application...'
                script {
                    def imageTag = "${env.BUILD_NUMBER}-${env.GIT_COMMIT.take(7)}"
                    sh """
                        docker build -t ${env.REGISTRY}/${env.IMAGE_NAME}:${imageTag} .
                        docker push ${env.REGISTRY}/${env.IMAGE_NAME}:${imageTag}
                    """
                }
            }
        }
        
        stage('Deploy') {
            when {
                branch 'main'
            }
            steps {
                echo '🚀 Deploying application...'
                // Add deployment commands here
            }
        }
    }
    
    post {
        always {
            cleanWs()
        }
        success {
            echo '✅ Pipeline completed successfully'
        }
        failure {
            echo '❌ Pipeline failed'
        }
    }
}
