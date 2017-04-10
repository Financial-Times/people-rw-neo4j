CREDENTIALS_DIR="credentials"
DOCKER_IMAGE_ID="coco/k8s-cli-utils"

node('docker') {
  catchError {
    stage "prepare credentials"
    prepareCredentials()

    stage "List pods"
    String currentDir = pwd()
    docker.image(DOCKER_IMAGE_ID).inside("-v ${currentDir}/${CREDENTIALS_DIR}:/${CREDENTIALS_DIR}") {
      sh "kubectl get pods"
    }
  }
  deleteDir()
}

public prepareCredentials() {
  withCredentials([
      [$class: 'FileBinding', credentialsId: 'ft.k8s.client-certificate', variable: 'CLIENT_CERT'],
      [$class: 'FileBinding', credentialsId: 'ft.k8s.ca-cert', variable: 'CA_CERT'],
      [$class: 'FileBinding', credentialsId: 'ft.k8s.client-key', variable: 'CLIENT_KEY']]) {
    sh """
      mkdir ${CREDENTIALS_DIR}
      cp ${env.CLIENT_CERT} ${CREDENTIALS_DIR}/
      cp ${env.CLIENT_KEY} ${CREDENTIALS_DIR}/
      cp ${env.CA_CERT} ${CREDENTIALS_DIR}/
    """
  }
}