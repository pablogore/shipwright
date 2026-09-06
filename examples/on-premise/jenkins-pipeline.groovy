// Jenkins Pipeline Example for Shipwright
// This example shows how to use Shipwright in an on-premise Jenkins environment

pipeline {
    agent any
    
    environment {
        // Version of Shipwright to use
        SHIPWRIGHT_VERSION = 'v1.0.0'
        
        // Registry configuration
        REGISTRY_URL = credentials('registry-url')
        REGISTRY_USERNAME = credentials('registry-username')
        REGISTRY_PASSWORD = credentials('registry-password')
        
        // Git configuration
        GIT_REPO_URL = "${env.GIT_URL}"
        GIT_BRANCH = "${env.BRANCH_NAME}"
        GIT_USER_EMAIL = credentials('git-user-email')
        GIT_USER_NAME = credentials('git-user-name')
        
        // Service configuration
        SERVICE_NAME = "${env.JOB_NAME}"
        SERVICE_VERSION = "${env.BUILD_NUMBER}"
        ENVIRONMENT = "${params.ENVIRONMENT ?: 'staging'}"
    }
    
    parameters {
        choice(
            name: 'ENVIRONMENT',
            choices: ['dev', 'staging', 'production'],
            description: 'Target environment'
        )
        choice(
            name: 'PIPELINE_TYPE',
            choices: ['go-service', 'docker-go', 'infra'],
            description: 'Pipeline type to execute'
        )
        booleanParam(
            name: 'SKIP_PUSH',
            defaultValue: false,
            description: 'Skip pushing to registry'
        )
    }
    
    stages {
        stage('Install Shipwright') {
            steps {
                script {
                    // Check if shipwright is already installed
                    def daggerInstalled = sh(
                        script: 'which shipwright || echo "not-found"',
                        returnStdout: true
                    ).trim()
                    
                    if (daggerInstalled == 'not-found') {
                        echo "📥 Installing Shipwright ${SHIPWRIGHT_VERSION}"
                        sh """
                            curl -L "https://github.com/pablogore/shipwright/releases/download/${SHIPWRIGHT_VERSION}/shipwright-linux-amd64" -o shipwright
                            chmod +x shipwright
                            sudo mv shipwright /usr/local/bin/
                        """
                    } else {
                        echo "✅ Shipwright already installed"
                    }
                    
                    // Verify installation
                    sh 'shipwright --version'
                }
            }
        }
        
        stage('Setup') {
            steps {
                echo "🔧 Running setup stage"
                sh """
                    shipwright \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=setup \
                        --config=.shipwright.yml \
                        --env=${params.ENVIRONMENT} \
                        --git-ref=${env.GIT_BRANCH} \
                        --git-auth=https \
                        --verbose
                """
            }
        }
        
        stage('Build') {
            steps {
                echo "🔨 Running build stage"
                sh """
                    shipwright \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=build \
                        --config=.shipwright.yml \
                        --env=${params.ENVIRONMENT} \
                        --verbose
                """
            }
            post {
                success {
                    archiveArtifacts artifacts: 'bin/**,dist/**', fingerprint: true
                }
            }
        }
        
        stage('Test') {
            steps {
                echo "🧪 Running test stage"
                sh """
                    shipwright \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=test \
                        --config=.shipwright.yml \
                        --env=${params.ENVIRONMENT} \
                        --coverage=90 \
                        --verbose
                """
            }
            post {
                always {
                    // Publish test results
                    publishTestResults testResultsPattern: 'test-results/**/*.xml'
                    
                    // Publish coverage reports
                    publishCoverageReport(
                        adapters: [
                            coberturaAdapter('coverage/coverage.xml')
                        ],
                        sourceFileResolver: sourceFiles('STORE_LAST_BUILD')
                    )
                }
            }
        }
        
        stage('Security Scan') {
            steps {
                echo "🔒 Running security scan"
                sh """
                    shipwright \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=security \
                        --config=.shipwright.yml \
                        --env=${params.ENVIRONMENT} \
                        --verbose
                """
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
                echo "📦 Running package stage"
                sh """
                    shipwright \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=package \
                        --config=.shipwright.yml \
                        --env=${params.ENVIRONMENT} \
                        --verbose
                """
            }
        }
        
        stage('Push to Registry') {
            when {
                allOf {
                    anyOf {
                        branch 'main'
                        branch 'develop'
                    }
                    expression { !params.SKIP_PUSH }
                }
            }
            steps {
                echo "🚀 Pushing to registry"
                sh """
                    shipwright \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=push \
                        --config=.shipwright.yml \
                        --env=${params.ENVIRONMENT} \
                        --verbose
                """
            }
        }
    }
    
    post {
        always {
            // Cleanup
            cleanWs()
        }
        success {
            echo "✅ Pipeline completed successfully"
        }
        failure {
            echo "❌ Pipeline failed"
            // Send notification
            emailext (
                subject: "Pipeline Failed: ${env.JOB_NAME} - ${env.BUILD_NUMBER}",
                body: "Pipeline failed. Check console output: ${env.BUILD_URL}",
                to: "${env.CHANGE_AUTHOR_EMAIL}"
            )
        }
    }
}

