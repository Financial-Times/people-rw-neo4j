@Library('k8s-pipeline-lib') _

import com.ft.jenkins.BuildConfig
import com.ft.jenkins.Cluster

BuildConfig config = new BuildConfig()
config.deployToClusters = [Cluster.DELIVERY]

entryPointForReleaseAndDev(config)