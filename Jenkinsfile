BASE_IMAGE_ID = "coco/people-rw-neo4j:"

node('docker') {
  stage 'checkout'
  checkout scm

  stage 'build-image'
  imgFullName = BASE_IMAGE_ID + getFeatureName(env.BRANCH_NAME)
  docker.build(imgFullName, ".")
  docker.withRegistry("https://hub.docker.com/", 'ft.dh.credentials') {
    docker.push(imgFullName)
  }

  deleteDir()
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

