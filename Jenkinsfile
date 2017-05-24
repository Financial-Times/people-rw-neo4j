@Library('k8s-pipeline-lib') _

import com.ft.up.BuildConfig
import com.ft.up.Cluster

BuildConfig config = new BuildConfig()
config.deployToClusters = [Cluster.DELIVERY]

entryPointForReleaseAndDev(config)