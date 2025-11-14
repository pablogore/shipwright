// Jenkins Pipeline Example for Syntegrity Dagger
// This example shows how to use Syntegrity Dagger in an on-premise Jenkins environment

pipeline {
    agent any
    
    environment {
        // Version of Syntegrity Dagger to use
        SYNTEGRITY_DAGGER_VERSION = 'v1.0.0'
        
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
            choices: ['go-kit', 'docker-go', 'infra'],
            description: 'Pipeline type to execute'
        )
        booleanParam(
            name: 'SKIP_PUSH',
            defaultValue: false,
            description: 'Skip pushing to registry'
        )
    }
    
    stages {
        stage('Install Syntegrity Dagger') {
            steps {
                script {
                    // Check if syntegrity-dagger is already installed
                    def daggerInstalled = sh(
                        script: 'which syntegrity-dagger || echo "not-found"',
                        returnStdout: true
                    ).trim()
                    
                    if (daggerInstalled == 'not-found') {
                        echo "📥 Installing Syntegrity Dagger ${SYNTEGRITY_DAGGER_VERSION}"
                        sh """
                            curl -L "https://github.com/getsyntegrity/syntegrity-dagger/releases/download/${SYNTEGRITY_DAGGER_VERSION}/syntegrity-dagger-linux-amd64" -o syntegrity-dagger
                            chmod +x syntegrity-dagger
                            sudo mv syntegrity-dagger /usr/local/bin/
                        """
                    } else {
                        echo "✅ Syntegrity Dagger already installed"
                    }
                    
                    // Verify installation
                    sh 'syntegrity-dagger --version'
                }
            }
        }
        
        stage('Setup') {
            steps {
                echo "🔧 Running setup stage"
                sh """
                    syntegrity-dagger \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=setup \
                        --config=.syntegrity-dagger.yml \
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
                    syntegrity-dagger \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=build \
                        --config=.syntegrity-dagger.yml \
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
                    syntegrity-dagger \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=test \
                        --config=.syntegrity-dagger.yml \
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
                    syntegrity-dagger \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=security \
                        --config=.syntegrity-dagger.yml \
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
                    syntegrity-dagger \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=package \
                        --config=.syntegrity-dagger.yml \
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
                    syntegrity-dagger \
                        --pipeline=${params.PIPELINE_TYPE} \
                        --step=push \
                        --config=.syntegrity-dagger.yml \
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

