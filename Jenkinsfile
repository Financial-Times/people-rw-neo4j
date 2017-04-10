BASE_IMAGE_ID = "coco/people-rw-neo4j:"

node('docker') {
  stage 'checkout'
  checkout scm

  stage 'build image'
  String imgFullName = BASE_IMAGE_ID + getFeatureName(env.BRANCH_NAME)
  def image = docker.build(imgFullName, ".")

  stage 'push image'
  docker.withRegistry("", 'ft.dh.credentials') {
    image.push()
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

