node('docker') {
  stage 'checkout'
  checkout scm
  
  stage 'build-image'
  docker.build("coco/people-rw-neo4j:pipeline01", ".") 
}