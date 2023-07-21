import groovy.json.JsonSlurper

pipeline {
  agent {
    label "exp-builder"
  }

  triggers {
    cron('H 2 * * *')
  }

  options {
    timeout(time: 20, unit: 'MINUTES')
    timestamps()
    buildDiscarder(logRotator(daysToKeepStr: '30'))
  }

  stages {
    stage('format') {
      steps {
        script {
          def dockerImage = docker.build("build", "./docker")
          dockerImage.inside {
            sh '''
              export GOPATH=/tmp/go
              export GOCACHE=/tmp/gocache
              test -z $(gofmt -l .)
            '''
          }
        }
      }
    }

    stage('validate') {
      steps {
        script {
          def dockerImage = docker.build("build", "./docker")
          dockerImage.inside {
            sh '''
              export GOPATH=/tmp/go
              export GOCACHE=/tmp/gocache
              go run . validate --strict
            '''
          }
        }
      }
    }
    
    stage('testsWithoutClang') {
      steps {
        script {
          def dockerImage = docker.build("build", "./docker")
          dockerImage.inside {
            sh '''
              export GOPATH=/tmp/go
              export GOCACHE=/tmp/gocache
              go test -v ./...
            '''
          }
        }
      }
    }

    stage('testsWithClang') {
      steps {
        script {
          def dockerImage = docker.build("build", "./docker")
          dockerImage.inside {
            sh '''
              mkdir .tools
              wget https://github.com/daedaleanai/llvm-project/releases/download/ddln-llvm14-rc/llvm-14.0.0.tar.gz
              tar -xf llvm-14.0.0.tar.gz -C .tools

              export GOPATH=/tmp/go
              export GOCACHE=/tmp/gocache
              export CGO_LDFLAGS="-L$PWD/.tools/llvm/lib -Wl,-rpath=$PWD/.tools/llvm/lib"
              go test --tags clang -v ./...
            '''
          }
        }
      }
    }

  }

  post {
    cleanup {
      cleanWs(disableDeferredWipeout: true, notFailBuild: true)
    }
  }
}
