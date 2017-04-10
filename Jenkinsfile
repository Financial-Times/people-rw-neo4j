BASE_IMAGE_ID = "coco/people-rw-neo4j:"
CREDENTIALS_DIR="credentials"
DOCKER_IMAGE_ID="coco/k8s-cli-utils"

node('docker') {
  catchError {
    stage 'checkout'
    checkout scm

    stage 'build image'
    String imageVersion = getFeatureName(env.BRANCH_NAME)
    String imgFullName = BASE_IMAGE_ID + imageVersion
    def image = docker.build(imgFullName, ".")

    stage 'push image'
    docker.withRegistry("", 'ft.dh.credentials') {
      image.push()
    }

    stage "deploy to team env"
    prepareCredentials()
    String currentDir = pwd()

    docker.image(DOCKER_IMAGE_ID).inside("-v ${currentDir}/${CREDENTIALS_DIR}:/${CREDENTIALS_DIR}") {
      sh "helm list"
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

String getEnvironment(String branchName) {
  String[] values = branchName.split('/')
  if (values.length < 3) {
    return ""
  }
  return values[1]
}

String getFeatureName(String branchName) {
  String[] values = branchName.split('/')
  if (values.length < 3) {
    return ""
  }
  return values[2]
}

