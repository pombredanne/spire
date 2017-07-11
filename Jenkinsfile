node  {
    def service = "spire"
    def image

    stage("Checkout") {
        checkout([
            $class: 'GitSCM',
            branches: scm.branches,
            doGenerateSubmoduleConfigurations: scm.doGenerateSubmoduleConfigurations,
            userRemoteConfigs: scm.userRemoteConfigs,
            extensions: scm.extensions + [[$class: 'SubmoduleOption', disableSubmodules: false, parentCredentials: true, recursiveSubmodules: true, reference: '', trackingSubmodules: false]]
        ])
        vers = sh(returnStdout: true, script: 'git describe --tags').trim()
    }
    stage("Build docker") {
        image = docker.build("superscale/${service}")
    }
    stage("Run Tests") {
        image.run('', '"/go/bin/ginkgo" "-r" "-cover"')
    }
}
