CREDENTIALS_DIR="credentials"
DOCKER_IMAGE_ID="coco/k8s-cli-utils:latest"
APP_NAME='people-rw-neo4j'

node {
  catchError {
    stage 'checkout'

    echo "Checking out tag $GIT_TAG"
    checkout([$class: 'GitSCM', branches: [[name: "refs/tags/$GIT_TAG"]], doGenerateSubmoduleConfigurations: false, extensions: [], submoduleCfg: [], userRemoteConfigs: [[url: 'https://github.com/Financial-Times/people-rw-neo4j']]])

    stage "prepare credentials"
    prepareCredentials()

    stage 'build-image'
    DOCKER_TAG = "coco/${APP_NAME}:${GIT_TAG}"
    echo "Building image $DOCKER_TAG"
    docker.build("coco/${APP_NAME}:pipeline${GIT_TAG}", ".")

    stage 'push-image'
    echo "TODO Push the image to dockerhub"

    stage 'CR for PRE-PROD'
    echo "TODO CR for PRE-PROD"

    stage 'deploy-to-pre-prod'
    String currentDir = pwd()
    docker.image(DOCKER_IMAGE_ID).inside("-v ${currentDir}/${CREDENTIALS_DIR}:/${CREDENTIALS_DIR}") {
      sh "kubectl set image deployments/${APP_NAME} ${APP_NAME}=\"coco/${APP_NAME}:v${GIT_TAG}\""
    }

    stage 'Validate in PRE-PROD'
    input message: 'Check the app in pre-prod', ok: 'App is ok in PRE-PROD'

    stage 'Deploy to PROD'
    input message: 'Press the button to deploy to prod', ok: 'Deploy to PROD'

    stage 'CR for PROD'
    echo "TODO CR for PROD"

    stage 'deploy-to-prod'
    docker.image(DOCKER_IMAGE_ID).inside("-v ${currentDir}/${CREDENTIALS_DIR}:/${CREDENTIALS_DIR}") {
      sh "kubectl set image deployments/${APP_NAME} ${APP_NAME}=\"coco/${APP_NAME}:v${GIT_TAG}\""
    }

    stage 'Validate in PROD'
    input message: 'Check the app in PROD', ok: 'App is OK in PROD'
  }
  deleteDir() 
}

public prepareCredentials() {
  withCredentials([
      [$class: 'FileBinding', credentialsId: 'ft.k8s.client-certificate', variable: 'CLIENT_CERT'],
      [$class: 'FileBinding', credentialsId: 'ft.k8s.ca-cert', variable: 'CA_CERT'],
      [$class: 'FileBinding', credentialsId: 'ft.k8s.client-key', variable: 'CLIENT_KEY']]) {
    sh """
      mkdir -p ${CREDENTIALS_DIR}
      cp ${env.CLIENT_CERT} ${CREDENTIALS_DIR}/
      cp ${env.CLIENT_KEY} ${CREDENTIALS_DIR}/
      cp ${env.CA_CERT} ${CREDENTIALS_DIR}/
    """
  }
}