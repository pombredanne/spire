node  {
    sh("rm -f ~/.dockercfg")

    def service = "spire"
    def image

    stage("Checkout") {
        checkout([
            $class: 'GitSCM',
            branches: scm.branches,
            doGenerateSubmoduleConfigurations: scm.doGenerateSubmoduleConfigurations,
            userRemoteConfigs: scm.userRemoteConfigs,
            extensions: scm.extensions + [[$class: 'SubmoduleOption', disableSubmodules: false, parentCredentials: true, recursiveSubmodules: true, reference: '', trackingSubmodules: false], [$class: 'CloneOption', depth: 0, noTags: false, reference: '', shallow: false]]
        ])
    }
    stage("Build Docker Image") {
        image = docker.build("haikoschol/${service}")
    }
    stage("Run Tests") {
        image.run('', '"/go/bin/ginkgo" "-r" "-cover" "-race"')
    }
    if (env.BRANCH_NAME == "master") {
        stage("Publish Docker Image") {
            def vers = sh(returnStdout: true, script: 'git describe --tags').trim()
            image.tag "${vers}"
            image.tag "latest"

            withCredentials([string(credentialsId: 'spire-docker-hub', variable: 'DOCKER_HUB_API_KEY')]) {
                sh("docker login -u haikoschol -p $DOCKER_HUB_API_KEY")
                sh("docker push haikoschol/${service}:${vers}")
                sh("docker push haikoschol/${service}:latest")
            }
        }
    }
}
