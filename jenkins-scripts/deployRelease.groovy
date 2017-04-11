CREDENTIALS_DIR = "credentials"
DOCKER_IMAGE_ID = "coco/k8s-cli-utils:latest"
APP_NAME = 'people-rw-neo4j'

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
        def dockerImg = docker.build("coco/${APP_NAME}:pipeline${GIT_TAG}", ".")

        stage 'push-image'
        echo "Pushing image ${DOCKER_TAG} to dockerhub"
        docker.withRegistry("", 'ft.dh.credentials') {
          dockerImg.push()
        }

        stage 'Open CR for PRE-PROD'
        echo "Opening CR for deployment to PRE-PROD."
        String gitTempFile = "tagDescription.txt"
        sh "git show ${GIT_TAG}  > ${gitTempFile}"
        String fileText = readFile(gitTempFile)
        string authorEmailAddress = getAuthorEmailAddress(fileText)
        string summaryOfChange = getSummaryOfChange(fileText)
        callJenkinsJobForOpeningCR(authorEmailAddress, summaryOfChange, "PRE-PROD")


        stage 'deploy-to-pre-prod'
        String currentDir = pwd()
        docker.image(DOCKER_IMAGE_ID).inside("-v ${currentDir}/${CREDENTIALS_DIR}:/${CREDENTIALS_DIR}") {
            sh "kubectl get pods --selector=app=${APP_NAME} -o jsonpath='{\$.items[0].spec.containers[*].image}' > image-version"
            echo "pre-prod old version: " + readFile("image-version")

            sh "kubectl set image deployments/${APP_NAME} ${APP_NAME}=\"coco/${APP_NAME}:pipeline${GIT_TAG}\""

            sh "kubectl get pods --selector=app=${APP_NAME} -o jsonpath='{\$.items[0].spec.containers[*].image}' > image-version"
            echo "pre-prod new version: " + readFile("image-version")
        }

        stage 'Close CR for PRE-PROD'
        echo "Closing CR for deployment to PRE-PROD."
        callJenkinsJobForClosingCR(authorEmailAddress, summaryOfChange, "PRE-PROD","CR1234")

        stage 'Validate in PRE-PROD'
        echo "Starting manual validation in PRE-PROD"
        withCredentials([[$class: 'StringBinding', credentialsId: 'ft.slack.hook', variable: 'SLACK_HOOK']]) {
            sh "curl -X POST --data-urlencode 'payload={\"username\": \"${APP_NAME} ${GIT_TAG} release\", \"text\": \"Manual action: <${env.BUILD_URL}input|click here to validate deployment to PRE-PROD>\", \"icon_emoji\": \":k8s:\"}' ${env.SLACK_HOOK}"
        }

        withCredentials([[$class: 'StringBinding', credentialsId: 'ft.k8s.preprod_env', variable: 'PRE_PROD_ENV']]) {
            input message: "Check the app in pre-prod: ${env.PRE_PROD_ENV}/__health/__pods-health?service-name=${APP_NAME}", ok: 'App is OK in PRE-PROD'
        }

        stage 'Deploy to PROD'
        withCredentials([[$class: 'StringBinding', credentialsId: 'ft.slack.hook', variable: 'SLACK_HOOK']]) {
            sh "curl -X POST --data-urlencode 'payload={\"username\": \"${APP_NAME} ${GIT_TAG} release\", \"text\": \"Manual action: <${env.BUILD_URL}input|click here to deploy to PROD>\", \"icon_emoji\": \":k8s:\"}' ${env.SLACK_HOOK}"
        }
        input message: 'Press the button to deploy to prod', ok: 'Deploy to PROD'

        stage 'Open CR for PROD'
        echo "Opening CR for deployment to PRE-PROD."
        callJenkinsJobForOpeningCR(authorEmailAddress, summaryOfChange, "PROD")

        stage 'deploy-to-prod'
        docker.image(DOCKER_IMAGE_ID).inside("-v ${currentDir}/${CREDENTIALS_DIR}:/${CREDENTIALS_DIR}") {
            sh "kubectl get pods --selector=app=${APP_NAME} -o jsonpath='{\$.items[0].spec.containers[*].image}' > image-version"
            echo "prod old version: " + readFile("image-version")

            sh "kubectl set image deployments/${APP_NAME} ${APP_NAME}=\"coco/${APP_NAME}:pipeline${GIT_TAG}\""

            sh "kubectl get pods --selector=app=${APP_NAME} -o jsonpath='{\$.items[0].spec.containers[*].image}' > image-version"
            echo "prod new version: " + readFile("image-version")
        }

        stage 'Close CR for PROD'
        echo "Closing CR for deployment to PRE-PROD."
        callJenkinsJobForClosingCR(authorEmailAddress, summaryOfChange, "PROD","CR1234")

        stage 'Validate in PROD'
        echo "Starting manual validation in PROD"
        withCredentials([[$class: 'StringBinding', credentialsId: 'ft.slack.hook', variable: 'SLACK_HOOK']]) {
            sh "curl -X POST --data-urlencode 'payload={\"username\": \"${APP_NAME} ${GIT_TAG} release\", \"text\": \"Manual action: <${env.BUILD_URL}input|click here to validate deployment to PROD>\", \"icon_emoji\": \":k8s:\"}' ${env.SLACK_HOOK}"
        }
        withCredentials([[$class: 'StringBinding', credentialsId: 'ft.k8s.prod_env', variable: 'PROD_ENV']]) {
            input message: "Check the app in PROD ${env.PROD_ENV}/__health/__pods-health?service-name=${APP_NAME}", ok: 'App is OK in PROD'
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
      mkdir -p ${CREDENTIALS_DIR}
      cp ${env.CLIENT_CERT} ${CREDENTIALS_DIR}/
      cp ${env.CLIENT_KEY} ${CREDENTIALS_DIR}/
      cp ${env.CA_CERT} ${CREDENTIALS_DIR}/
    """
    }
}

@NonCPS public String getAuthorEmailAddress(String fileText) {
  return (fileText =~ /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[A-Za-z]{2,4}/)[0]
}

@NonCPS public String getSummaryOfChange(String fileText) {
  String description = ""
  fileText.eachLine { line ->
    if (!(line =~ /Author:/) && !(line =~ /Date:/)  && !(line =~ /Merge:/) && line!="") {
      description = description+line
    }
  }

  return description
}

public callJenkinsJobForOpeningCR(String emailAddress, String summaryOfChange, String envName) {
  echo "Calling jenkins job for opening CR with params: email: [${emailAddress}] summary: [${summaryOfChange}] environment: [${envName}]"

}

public callJenkinsJobForClosingCR(String emailAddress, String summaryOfChange, String envName,String id) {
  echo "Calling jenkins job for closing CR with params: email: [${emailAddress}] summary: [${summaryOfChange}] environment: [${envName}] and id: [${id}]"
}
