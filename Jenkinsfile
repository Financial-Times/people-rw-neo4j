@Library('k8s-pipeline-lib@first-version') _

import com.ft.up.BuildConfig

BuildConfig config = new BuildConfig()
config.appDockerImageId = "coco/people-rw-neo4j"
config.useInternalDockerReg = false

entryPointForReleaseAndDev(config)