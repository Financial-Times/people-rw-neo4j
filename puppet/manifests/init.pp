class people_rw_neo4j {

  $configParameters = hiera('configParameters','')

  class { "go_service_profile" :
    service_module => $module_name,
    service_name => "people-rw-neo4j",
    configParameters => $configParameters
  }
}
