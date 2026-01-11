# =============================================================================
# VERIFICATION CHECKS - Symlink resource validation
# =============================================================================

check "verify_basic_symlink" {
  data "filemanager_stat" "basic_symlink_check" {
    path = filemanager_symlink.basic_absolute.path
  }

  assert {
    condition     = data.filemanager_stat.basic_symlink_check.exists == true
    error_message = "Basic symlink does not exist"
  }

  assert {
    condition     = data.filemanager_stat.basic_symlink_check.is_symlink == true
    error_message = "Path should be a symlink"
  }
}

check "verify_dir_symlink" {
  data "filemanager_stat" "dir_symlink_check" {
    path = filemanager_symlink.dir_absolute.path
  }

  assert {
    condition     = data.filemanager_stat.dir_symlink_check.exists == true
    error_message = "Directory symlink does not exist"
  }

  assert {
    condition     = data.filemanager_stat.dir_symlink_check.is_symlink == true
    error_message = "Path should be a symlink"
  }
}

check "verify_relative_symlink" {
  data "filemanager_stat" "relative_check" {
    path = filemanager_symlink.relative_same_dir.path
  }

  assert {
    condition     = data.filemanager_stat.relative_check.exists == true
    error_message = "Relative symlink does not exist"
  }

  assert {
    condition     = data.filemanager_stat.relative_check.is_symlink == true
    error_message = "Path should be a symlink"
  }
}

check "verify_symlink_chain_1" {
  data "filemanager_stat" "chain1_check" {
    path = filemanager_symlink.chain_1.path
  }

  assert {
    condition     = data.filemanager_stat.chain1_check.is_symlink == true
    error_message = "Chain link 1 should be a symlink"
  }
}

check "verify_symlink_chain_2" {
  data "filemanager_stat" "chain2_check" {
    path = filemanager_symlink.chain_2.path
  }

  assert {
    condition     = data.filemanager_stat.chain2_check.is_symlink == true
    error_message = "Chain link 2 should be a symlink"
  }
}

check "verify_symlink_chain_3" {
  data "filemanager_stat" "chain3_check" {
    path = filemanager_symlink.chain_3.path
  }

  assert {
    condition     = data.filemanager_stat.chain3_check.is_symlink == true
    error_message = "Chain link 3 should be a symlink"
  }
}

check "verify_multi_links" {
  data "filemanager_stat" "multi1_check" {
    path = filemanager_symlink.multi_link_1.path
  }

  assert {
    condition     = data.filemanager_stat.multi1_check.is_symlink == true
    error_message = "Multi link 1 should be a symlink"
  }
}

check "verify_hidden_symlink" {
  data "filemanager_stat" "hidden_symlink_check" {
    path = filemanager_symlink.hidden.path
  }

  assert {
    condition     = data.filemanager_stat.hidden_symlink_check.exists == true
    error_message = "Hidden symlink does not exist"
  }

  assert {
    condition     = data.filemanager_stat.hidden_symlink_check.is_symlink == true
    error_message = "Hidden path should be a symlink"
  }
}

check "verify_deep_to_shallow" {
  data "filemanager_stat" "deep_shallow_check" {
    path = filemanager_symlink.deep_to_shallow.path
  }

  assert {
    condition     = data.filemanager_stat.deep_shallow_check.is_symlink == true
    error_message = "Deep to shallow link should be a symlink"
  }
}

# Verify symlink content is accessible
check "verify_symlink_content" {
  data "filemanager_file" "through_symlink" {
    path = filemanager_symlink.basic_absolute.path
  }

  assert {
    condition     = data.filemanager_file.through_symlink.content == "This is the target file"
    error_message = "Content through symlink should match target file"
  }
}
