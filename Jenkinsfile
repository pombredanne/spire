node  {
    sh("rm ~/.dockercfg")
    env.AWS_ECR_LOGIN=true

    docker.withRegistry("https://531572303926.dkr.ecr.eu-west-1.amazonaws.com/superscale/", "ecr:eu-west-1:aws") {
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
        }
        stage("Build docker") {
            image = docker.build("superscale/${service}")
        }
        stage("Run Tests") {
            image.run('', '"/go/bin/ginkgo" "-r" "-cover"')
        }
        if (env.BRANCH_NAME == "master") {
            withCredentials([$class: 'AmazonWebServicesCredentialsBinding', accessKeyVariable: 'AWS_ACCESS_KEY_ID', credentialsId: 'aws', secretKeyVariable: 'AWS_SECRET_ACCESS_KEY']) {
        	   stage("Publish ECR") {
                    def vers = sh(returnStdout: true, script: 'git describe --tags').trim()
                	image.push "${vers}"
        	   }
            }
        }
    }
}
