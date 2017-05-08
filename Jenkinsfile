devBuildAndDeploy(this, "coco/people-rw-neo4j")

CREDENTIALS_DIR = "credentials"
DOCKER_IMAGE_ID = "coco/k8s-cli-utils:update-helm"

envToApiServerMap = [
    "xp"  : "https://k8s-delivery-upp-eu-api.ft.com",
    "test": "https://k8s-delivery-upp-eu-api.ft.com"
]

HELM_CONFIG_FOLDER="helm"

node('docker') {
  catchError {
    stage('checkout') {
      checkout scm
    }

    String imageVersion = getFeatureName(env.BRANCH_NAME)
    stage('build image') {
      buildImage(imageVersion)
    }

    String env = getEnvironment(env.BRANCH_NAME)
    //  todo [sb] handle the case when the environment is not specified in the branch name

    stage("deploy to team env") {
      deployAppWithHelm(imageVersion, env)
    }
  }

  deleteDir()
}

public deployAppWithHelm(String imageVersion, String env) {
  runWithK8SCliTools(env) {
    def chartName = getHelmChartFolderName()
    /*  todo [sb] handle the case when the chart is used by more than 1 app */
    /*  using the chart name also as release name.. we have one release per app */
    sh "helm upgrade ${chartName} ${HELM_CONFIG_FOLDER}/${chartName} -i --set image.version=${imageVersion}"
  }
}

/**
 * Retrieves the folder name where the Helm chart is defined .
 *
 * @return
 */
public String getHelmChartFolderName() {
  org.jenkinsci.plugins.pipeline.utility.steps.fs.FileWrapper chartFile = findFiles(glob: "${HELM_CONFIG_FOLDER}/**/Chart.yaml")[0]
  String[] chartFilePathComponents = ((String) chartFile.path).split('/')
  /* return the parent folder of Chart.yaml */
  return chartFilePathComponents[chartFilePathComponents.size() - 2]
}

public void runWithK8SCliTools(String env, Closure codeToRun) {
  prepareCredentials()
  String currentDir = pwd()

  String apiServer = getApiServerForEnvironment(env)
  GString dockerRunArgs =
      "-v ${currentDir}/${CREDENTIALS_DIR}:/${CREDENTIALS_DIR} " +
      "-e 'K8S_API_SERVER=${apiServer}' " +
      "-e 'KUBECONFIG=${currentDir}/kubeconfig'"

  docker.image(DOCKER_IMAGE_ID).inside(dockerRunArgs) {
    sh "/docker-entrypoint.sh"

    codeToRun.call()
  }
}

public String getApiServerForEnvironment(String envName) {
  String apiServer = envToApiServerMap[envName]
  if (apiServer) {
    return apiServer
  }
  /*  return this for now, as it is our only cluster */
  return "https://k8s-delivery-upp-eu-api.ft.com"
}

public void pushImageToDH(image) {
  docker.withRegistry("", 'ft.dh.credentials') {
    image.push()
  }
}

public void buildImage(String imageVersion) {
  String imgFullName = BASE_IMAGE_ID + imageVersion
  def image = docker.build(imgFullName, ".")
  pushImageToDH(image)
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

/**
 * Gets the environment name where to deploy from the specified branch name by getting the penultimate one path item.
 * <p>
 * Example:
 * <ol>
 *   <li> for a branch named "feature/xp/test", it will return "xp".</li>
 *   <li> for a branch named "test", it will return null.</li>
 * </ol>
 * @param branchName the name of the branch
 * @return the environment name where to deploy the branch
 */
String getEnvironment(String branchName) {
  String[] values = branchName.split('/')
  if (values.length > 2) {
    return values[values.length - 2]
  }
  return null
}

/**
 * Gets the feature name from a branch name by getting the last item after the last "/".
 * Example: for a branch name as "feature/xp/test", it will return "test".
 *
 * @param branchName the name of the branch
 * @return the feature name
 */
String getFeatureName(String branchName) {
  String[] values = branchName.split('/')
  return values[values.length - 1]
}

