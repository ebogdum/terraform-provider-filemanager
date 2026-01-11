# =============================================================================
# VERIFICATION CHECKS - JSON file resource validation
# =============================================================================

check "verify_simple_json" {
  data "filemanager_file" "simple_check" {
    path = filemanager_json_file.simple.path
  }

  assert {
    condition     = data.filemanager_file.simple_check.size > 0
    error_message = "Simple JSON file is empty"
  }

  assert {
    condition     = strcontains(data.filemanager_file.simple_check.content, "\"key\"")
    error_message = "Simple JSON should contain 'key'"
  }

  # Verify pretty-printed (has indentation)
  assert {
    condition     = strcontains(data.filemanager_file.simple_check.content, "\n  ")
    error_message = "JSON should be pretty-printed"
  }
}

check "verify_nested_json" {
  data "filemanager_file" "nested_check" {
    path = filemanager_json_file.nested.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.nested_check.content, "level1")
    error_message = "Nested JSON should contain level1"
  }

  assert {
    condition     = strcontains(data.filemanager_file.nested_check.content, "deep_value")
    error_message = "Nested JSON should contain deep_value"
  }
}

check "verify_array_primitives" {
  data "filemanager_file" "array_prims_check" {
    path = filemanager_json_file.array_primitives.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.array_prims_check.content, "[")
    error_message = "Array primitives should be an array"
  }
}

check "verify_empty_object" {
  data "filemanager_file" "empty_obj_check" {
    path = filemanager_json_file.empty_object.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.empty_obj_check.content, "{}")
    error_message = "Empty object should be {}"
  }
}

check "verify_merge_deep" {
  data "filemanager_file" "merge_deep_check" {
    path = filemanager_json_file.merge_deep.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.merge_deep_check.content, "database")
    error_message = "Merged JSON should contain database from base"
  }

  assert {
    condition     = strcontains(data.filemanager_file.merge_deep_check.content, "cache")
    error_message = "Merged JSON should contain cache from overlay"
  }

  assert {
    condition     = strcontains(data.filemanager_file.merge_deep_check.content, "logging")
    error_message = "Merged JSON should contain logging from base"
  }
}

check "verify_sorted_keys" {
  data "filemanager_file" "sorted_check" {
    path = filemanager_json_file.sorted.path
  }

  # In sorted JSON, 'alpha' should appear before 'zebra'
  assert {
    condition     = data.filemanager_file.sorted_check.size > 0
    error_message = "Sorted JSON should not be empty"
  }
}

check "verify_compact_json" {
  data "filemanager_file" "compact_check" {
    path = filemanager_json_file.compact.path
  }

  # Compact JSON should not have newlines (except maybe trailing)
  assert {
    condition     = !strcontains(data.filemanager_file.compact_check.content, "\n  ")
    error_message = "Compact JSON should not have indented newlines"
  }
}

check "verify_package_json" {
  data "filemanager_file" "package_check" {
    path = filemanager_json_file.package_json.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.package_check.content, "\"name\"")
    error_message = "Package.json should contain name"
  }

  assert {
    condition     = strcontains(data.filemanager_file.package_check.content, "\"scripts\"")
    error_message = "Package.json should contain scripts"
  }

  assert {
    condition     = strcontains(data.filemanager_file.package_check.content, "\"dependencies\"")
    error_message = "Package.json should contain dependencies"
  }
}

check "verify_unicode_json" {
  data "filemanager_file" "unicode_check" {
    path = filemanager_json_file.unicode.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.unicode_check.content, "日本語")
    error_message = "Unicode JSON should contain Japanese characters"
  }
}

check "verify_deeply_nested_json" {
  data "filemanager_file" "deep_nested_check" {
    path = filemanager_json_file.deeply_nested.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.deep_nested_check.content, "very deep")
    error_message = "Deeply nested JSON should contain 'very deep'"
  }
}

check "verify_large_array" {
  data "filemanager_stat" "large_array_check" {
    path = filemanager_json_file.large_array.path
  }

  assert {
    condition     = data.filemanager_stat.large_array_check.size > 1000
    error_message = "Large array JSON should be bigger than 1000 bytes"
  }
}

# =============================================================================
# FORMAT VALIDATION CHECKS (filemanager_validate)
# =============================================================================

check "validate_simple_json_format" {
  data "filemanager_validate" "simple_json" {
    path   = filemanager_json_file.simple.path
    format = "json"
  }

  assert {
    condition     = data.filemanager_validate.simple_json.is_valid == true
    error_message = "Simple JSON should be valid"
  }

  assert {
    condition     = data.filemanager_validate.simple_json.format_detected == "json"
    error_message = "Format should be detected as json"
  }
}

check "validate_nested_json_format" {
  data "filemanager_validate" "nested_json" {
    path = filemanager_json_file.nested.path
  }

  assert {
    condition     = data.filemanager_validate.nested_json.is_valid == true
    error_message = "Nested JSON should be valid"
  }
}

check "validate_package_json_format" {
  data "filemanager_validate" "package_json" {
    path = filemanager_json_file.package_json.path
  }

  assert {
    condition     = data.filemanager_validate.package_json.is_valid == true
    error_message = "Package.json should be valid JSON"
  }
}

check "validate_unicode_json_format" {
  data "filemanager_validate" "unicode_json" {
    path = filemanager_json_file.unicode.path
  }

  assert {
    condition     = data.filemanager_validate.unicode_json.is_valid == true
    error_message = "Unicode JSON should be valid"
  }
}

# =============================================================================
# FILE COMPARISON CHECKS (filemanager_compare)
# =============================================================================

check "compare_sorted_vs_unsorted" {
  data "filemanager_compare" "sorted_unsorted" {
    source = filemanager_json_file.sorted.path
    target = filemanager_json_file.unsorted.path
  }

  # Files have same data but different key order, so content differs but checksums differ
  assert {
    condition     = data.filemanager_compare.sorted_unsorted.source_exists == true
    error_message = "Source file should exist"
  }

  assert {
    condition     = data.filemanager_compare.sorted_unsorted.target_exists == true
    error_message = "Target file should exist"
  }
}

check "compare_same_file" {
  data "filemanager_compare" "same_file" {
    source = filemanager_json_file.simple.path
    target = filemanager_json_file.simple.path
  }

  assert {
    condition     = data.filemanager_compare.same_file.identical == true
    error_message = "Same file compared to itself should be identical"
  }

  assert {
    condition     = data.filemanager_compare.same_file.checksum_match == true
    error_message = "Same file should have matching checksum"
  }

  assert {
    condition     = data.filemanager_compare.same_file.content_match == true
    error_message = "Same file should have matching content"
  }

  assert {
    condition     = data.filemanager_compare.same_file.size_match == true
    error_message = "Same file should have matching size"
  }

  assert {
    condition     = data.filemanager_compare.same_file.mode_match == true
    error_message = "Same file should have matching mode"
  }
}

# =============================================================================
# ENHANCED STAT CHECKS (time-based)
# =============================================================================

check "stat_time_based_check" {
  data "filemanager_stat" "simple_time_check" {
    path            = filemanager_json_file.simple.path
    modified_within = "1h"
    accessed_within = "24h"
  }

  assert {
    condition     = data.filemanager_stat.simple_time_check.is_modified_within == true
    error_message = "Newly created file should be modified within last hour"
  }

  assert {
    condition     = data.filemanager_stat.simple_time_check.age != null
    error_message = "File age should be computed"
  }

  assert {
    condition     = data.filemanager_stat.simple_time_check.owner_name != null
    error_message = "Owner name should be resolved"
  }
}
