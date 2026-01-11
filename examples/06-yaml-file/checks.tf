# =============================================================================
# VERIFICATION CHECKS - YAML file resource validation
# =============================================================================

check "verify_simple_yaml" {
  data "filemanager_file" "simple_check" {
    path = filemanager_yaml_file.simple.path
  }

  assert {
    condition     = data.filemanager_file.simple_check.size > 0
    error_message = "Simple YAML file is empty"
  }

  assert {
    condition     = strcontains(data.filemanager_file.simple_check.content, "key:")
    error_message = "Simple YAML should contain 'key:'"
  }
}

check "verify_nested_yaml" {
  data "filemanager_file" "nested_check" {
    path = filemanager_yaml_file.nested.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.nested_check.content, "database:")
    error_message = "Nested YAML should contain 'database:'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.nested_check.content, "host:")
    error_message = "Nested YAML should contain 'host:'"
  }
}

check "verify_list_yaml" {
  data "filemanager_file" "list_check" {
    path = filemanager_yaml_file.list.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.list_check.content, "fruits:")
    error_message = "List YAML should contain 'fruits:'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.list_check.content, "- apple") || strcontains(data.filemanager_file.list_check.content, "apple")
    error_message = "List YAML should contain 'apple'"
  }
}

check "verify_docker_simple" {
  data "filemanager_file" "docker_simple_check" {
    path = filemanager_yaml_file.docker_simple.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.docker_simple_check.content, "version:")
    error_message = "Docker compose should contain 'version:'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.docker_simple_check.content, "services:")
    error_message = "Docker compose should contain 'services:'"
  }
}

check "verify_docker_complex" {
  data "filemanager_file" "docker_complex_check" {
    path = filemanager_yaml_file.docker_complex.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.docker_complex_check.content, "frontend")
    error_message = "Complex docker compose should contain 'frontend' service"
  }
}

check "verify_k8s_deployment" {
  data "filemanager_file" "k8s_deploy_check" {
    path = filemanager_yaml_file.k8s_deployment.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.k8s_deploy_check.content, "apiVersion:")
    error_message = "K8s deployment should contain 'apiVersion:'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.k8s_deploy_check.content, "kind:")
    error_message = "K8s deployment should contain 'kind:'"
  }
}

check "verify_github_actions" {
  data "filemanager_file" "github_check" {
    path = filemanager_yaml_file.github_actions.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.github_check.content, "name:")
    error_message = "GitHub Actions should contain 'name:'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.github_check.content, "jobs:")
    error_message = "GitHub Actions should contain 'jobs:'"
  }
}

# =============================================================================
# FORMAT VALIDATION CHECKS (filemanager_validate)
# =============================================================================

check "validate_simple_yaml_format" {
  data "filemanager_validate" "simple_yaml" {
    path   = filemanager_yaml_file.simple.path
    format = "yaml"
  }

  assert {
    condition     = data.filemanager_validate.simple_yaml.is_valid == true
    error_message = "Simple YAML should be valid"
  }
}

check "validate_k8s_deployment_format" {
  data "filemanager_validate" "k8s_yaml" {
    path = filemanager_yaml_file.k8s_deployment.path
  }

  assert {
    condition     = data.filemanager_validate.k8s_yaml.is_valid == true
    error_message = "K8s deployment YAML should be valid"
  }
}

check "validate_docker_compose_format" {
  data "filemanager_validate" "docker_yaml" {
    path = filemanager_yaml_file.docker_complex.path
  }

  assert {
    condition     = data.filemanager_validate.docker_yaml.is_valid == true
    error_message = "Docker compose YAML should be valid"
  }
}

# =============================================================================
# FILE COMPARISON CHECKS (filemanager_compare)
# =============================================================================

check "compare_yaml_same_file" {
  data "filemanager_compare" "yaml_same" {
    source = filemanager_yaml_file.simple.path
    target = filemanager_yaml_file.simple.path
  }

  assert {
    condition     = data.filemanager_compare.yaml_same.identical == true
    error_message = "Same YAML file compared to itself should be identical"
  }
}

# =============================================================================
# ENHANCED STAT CHECKS (time-based)
# =============================================================================

check "stat_yaml_time_check" {
  data "filemanager_stat" "yaml_time_check" {
    path            = filemanager_yaml_file.simple.path
    modified_within = "1h"
  }

  assert {
    condition     = data.filemanager_stat.yaml_time_check.is_modified_within == true
    error_message = "Newly created YAML file should be modified within last hour"
  }

  assert {
    condition     = data.filemanager_stat.yaml_time_check.age != null
    error_message = "YAML file age should be computed"
  }
}
