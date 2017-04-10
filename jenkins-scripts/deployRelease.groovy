CREDENTIALS_DIR = "credentials"
SLACK_HOOK = "foobar"
DOCKER_IMAGE_ID = "coco/k8s-cli-utils:latest"
APP_NAME = 'people-rw-neo4j'
PRE_PROD_ENV = "foo-pre-prod"
PROD_ENV = "foo-prod"

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
        echo "TODO open CR for PRE-PROD"

        stage 'deploy-to-pre-prod'
        String currentDir = pwd()
        docker.image(DOCKER_IMAGE_ID).inside("-v ${currentDir}/${CREDENTIALS_DIR}:/${CREDENTIALS_DIR}") {
            sh "kubectl get pods --selector=app=topics-rw-neo4j -o jsonpath='{\$.items[0].spec.containers[*].image}' > image-version"
            echo "pre-prod old version: " + readFile("image-version")

            sh "kubectl set image deployments/${APP_NAME} ${APP_NAME}=\"coco/${APP_NAME}:v${GIT_TAG}\""

            sh "kubectl get pods --selector=app=topics-rw-neo4j -o jsonpath='{\$.items[0].spec.containers[*].image}' > image-version"
            echo "pre-prod new version: " + readFile("image-version")
        }

        stage 'Close CR for PRE-PROD'
        echo "Closing CR for deployment to PRE-PROD."
        echo "TODO close CR for PRE-PROD"

        stage 'Validate in PRE-PROD'
        echo "Starting manual validation in PRE-PROD"
        sh "curl -X POST --data-urlencode 'payload={\"username\": \"Jenkins\", \"text\": \"Manual action needed: <${env.BUILD_URL}input|click here to validate deployment to PRE-PROD>\", \"icon_emoji\": \":k8s:\"}' ${SLACK_HOOK}"
        input message: "Check the app in pre-prod: https://$PRE_PROD_ENV/__health/__pods-health?service-name=${APP_NAME}", ok: 'App is OK in PRE-PROD'

        stage 'Deploy to PROD'
        echo "TODO slack or email integration fro deployment to PROD"
        input message: 'Press the button to deploy to prod', ok: 'Deploy to PROD'

        stage 'Open CR for PROD'
        echo "Opening CR for deployment to PRE-PROD."
        echo "TODO CR for PROD"

        stage 'deploy-to-prod'
        docker.image(DOCKER_IMAGE_ID).inside("-v ${currentDir}/${CREDENTIALS_DIR}:/${CREDENTIALS_DIR}") {
            sh "kubectl get pods --selector=app=${APP_NAME} -o jsonpath='{\$.items[0].spec.containers[*].image}' > image-version"
            echo "prod old version: " + readFile("image-version")

            sh "kubectl set image deployments/${APP_NAME} ${APP_NAME}=\"coco/${APP_NAME}:v${GIT_TAG}\""

            sh "kubectl get pods --selector=app=${APP_NAME} -o jsonpath='{\$.items[0].spec.containers[*].image}' > image-version"
            echo "prod new version: " + readFile("image-version")
        }

        stage 'Close CR for PROD'
        echo "Closing CR for deployment to PRE-PROD."
        echo "TODO CR for PROD"

        stage 'Validate in PROD'
        echo "Starting manual validation in PROD"
        echo "TODO slack or email integration for deployment to PROD"
        input message: 'Check the app in PROD https://${PROD_ENV}/__health/__pods-health?service-name=${APP_NAME}', ok: 'App is OK in PROD'
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