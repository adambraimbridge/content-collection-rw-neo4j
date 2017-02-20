class content-collection-rw-neo4j {
  $configParameters = hiera('configParameters','')

  class { "go_service_profile" :
    service_module => $module_name,
    service_name => 'content-collection-rw-neo4j',
    configParameters => $configParameters
  }
}
