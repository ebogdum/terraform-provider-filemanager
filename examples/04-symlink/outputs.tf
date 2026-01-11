# =============================================================================
# SYMLINK RESOURCE - OUTPUTS
# =============================================================================

output "basic_symlinks" {
  description = "Basic symlink tests"
  value = {
    absolute_file = {
      path   = filemanager_symlink.basic_absolute.path
      target = filemanager_symlink.basic_absolute.target
    }
    absolute_dir = {
      path   = filemanager_symlink.dir_absolute.path
      target = filemanager_symlink.dir_absolute.target
    }
    relative_same = {
      path   = filemanager_symlink.relative_same_dir.path
      target = filemanager_symlink.relative_same_dir.target
    }
    relative_parent = {
      path   = filemanager_symlink.relative_parent.path
      target = filemanager_symlink.relative_parent.target
    }
    relative_nested = {
      path   = filemanager_symlink.relative_nested.path
      target = filemanager_symlink.relative_nested.target
    }
  }
}

output "multi_links" {
  description = "Multiple symlinks to same target"
  value = {
    link1 = filemanager_symlink.multi_link_1.path
    link2 = filemanager_symlink.multi_link_2.path
    link3 = filemanager_symlink.multi_link_3.path
  }
}

output "chain_links" {
  description = "Symlink chain"
  value = {
    chain1 = filemanager_symlink.chain_1.path
    chain2 = filemanager_symlink.chain_2.path
    chain3 = filemanager_symlink.chain_3.path
  }
}

output "special_name_links" {
  description = "Symlinks with special names"
  value = {
    spaces   = filemanager_symlink.spaces_in_name.path
    hidden   = filemanager_symlink.hidden.path
    with_ext = filemanager_symlink.with_extension.path
    no_ext   = filemanager_symlink.no_ext_to_ext.path
  }
}

output "summary" {
  description = "Test summary"
  value = {
    total_symlinks = 18
    categories = [
      "basic_absolute",
      "basic_relative",
      "multi_link",
      "chain",
      "special_names",
      "directory_links"
    ]
  }
}
