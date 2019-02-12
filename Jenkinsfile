pipeline {
    agent {
        label 't2medium'
    }

    options {
        disableConcurrentBuilds()
    }

    environment {
        HUB_URL  = credentials('docker_hub_url')
        HUB_PATH = credentials('docker_hub_path')
    }

    stages {
        stage('Prepare') {
            steps {
                sh 'cp config_test.yml.dist config_test.yml'
                compose 'up -d --build postgres_test'
            }
        }

        stage('Tests') {
            steps {
               compose 'run --rm mg_bot_test make jenkins_test'
            }

            post {
                always {
                    sh 'cat ./test-report.xml'
                    junit 'test-report.xml'
                }
            }
        }

        stage('Docker Images') {
            when {
                branch 'master'
            }
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'docker-hub-credentials',
                    usernameVariable: 'HUB_USER',
                    passwordVariable: 'HUB_PASS'
                )]) {
                    sh 'echo ${HUB_PASS} | docker login -u ${HUB_USER} --password-stdin ${HUB_URL}'
                }

                sh 'docker build -t ${HUB_URL}${HUB_PATH} ./'
                sh 'docker push ${HUB_URL}${HUB_PATH}'
            }
            post {
                always {
                    sh 'docker rmi ${HUB_URL}${HUB_PATH}:latest'
                }
            }
        }
    }

    post {
        always {
            compose 'down -v'
            deleteDir ()
        }
        aborted {
            echo "Aborted."
        }
        success {
            echo "Success."
        }
        failure {
            echo "Failure."
        }
    }
}

def compose(cmd) {
    sh "docker-compose --no-ansi -f docker-compose-test.yml ${cmd}"
}
