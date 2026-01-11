# COPY RESOURCE - OUTPUTS

# -----------------------------------------------------------------------------
# SINGLE FILE COPIES
# -----------------------------------------------------------------------------

output "single_file_copies" {
  value = {
    basic = {
      source       = filemanager_copy.single_basic.source
      destination  = filemanager_copy.single_basic.destination
      files_copied = filemanager_copy.single_basic.files_copied
      bytes_copied = filemanager_copy.single_basic.bytes_copied
      md5          = filemanager_copy.single_basic.md5
      sha256       = filemanager_copy.single_basic.sha256
    }
    renamed = {
      source       = filemanager_copy.single_rename.source
      destination  = filemanager_copy.single_rename.destination
      files_copied = filemanager_copy.single_rename.files_copied
    }
    executable = {
      source               = filemanager_copy.single_executable.source
      destination          = filemanager_copy.single_executable.destination
      preserve_permissions = filemanager_copy.single_executable.preserve_permissions
    }
    custom_perms = {
      source          = filemanager_copy.single_custom_perms.source
      destination     = filemanager_copy.single_custom_perms.destination
      file_permission = filemanager_copy.single_custom_perms.file_permission
    }
    large = {
      source       = filemanager_copy.single_large.source
      destination  = filemanager_copy.single_large.destination
      bytes_copied = filemanager_copy.single_large.bytes_copied
    }
  }
}

# -----------------------------------------------------------------------------
# DIRECTORY COPIES
# -----------------------------------------------------------------------------

output "directory_copies" {
  value = {
    basic = {
      source       = filemanager_copy.dir_basic.source
      destination  = filemanager_copy.dir_basic.destination
      files_copied = filemanager_copy.dir_basic.files_copied
      bytes_copied = filemanager_copy.dir_basic.bytes_copied
    }
    with_excludes = {
      source       = filemanager_copy.dir_excludes.source
      destination  = filemanager_copy.dir_excludes.destination
      excludes     = filemanager_copy.dir_excludes.excludes
      files_copied = filemanager_copy.dir_excludes.files_copied
    }
    multi_excludes = {
      source       = filemanager_copy.dir_multi_excludes.source
      destination  = filemanager_copy.dir_multi_excludes.destination
      excludes     = filemanager_copy.dir_multi_excludes.excludes
      files_copied = filemanager_copy.dir_multi_excludes.files_copied
    }
    preserved_perms = {
      source               = filemanager_copy.dir_preserve_perms.source
      destination          = filemanager_copy.dir_preserve_perms.destination
      preserve_permissions = filemanager_copy.dir_preserve_perms.preserve_permissions
    }
    custom_perms = {
      source               = filemanager_copy.dir_custom_perms.source
      destination          = filemanager_copy.dir_custom_perms.destination
      file_permission      = filemanager_copy.dir_custom_perms.file_permission
      directory_permission = filemanager_copy.dir_custom_perms.directory_permission
    }
  }
}

# -----------------------------------------------------------------------------
# OVERWRITE SCENARIOS
# -----------------------------------------------------------------------------

output "overwrite_scenarios" {
  value = {
    enabled = {
      destination = filemanager_copy.overwrite_enabled.destination
      overwrite   = filemanager_copy.overwrite_enabled.overwrite
    }
    disabled_new = {
      destination = filemanager_copy.overwrite_disabled_new.destination
      overwrite   = filemanager_copy.overwrite_disabled_new.overwrite
    }
  }
}

# -----------------------------------------------------------------------------
# NESTED COPIES
# -----------------------------------------------------------------------------

output "nested_copies" {
  value = {
    deep_dir = {
      source       = filemanager_copy.nested_deep.source
      destination  = filemanager_copy.nested_deep.destination
      files_copied = filemanager_copy.nested_deep.files_copied
    }
    single_file = {
      source      = filemanager_copy.nested_single.source
      destination = filemanager_copy.nested_single.destination
    }
  }
}

# -----------------------------------------------------------------------------
# MULTIPLE COPIES
# -----------------------------------------------------------------------------

output "multi_copies" {
  value = {
    copy1 = filemanager_copy.multi_copy_1.destination
    copy2 = filemanager_copy.multi_copy_2.destination
    copy3 = filemanager_copy.multi_copy_3.destination
  }
}

# -----------------------------------------------------------------------------
# SUBDIRECTORY COPIES
# -----------------------------------------------------------------------------

output "subdir_copies" {
  value = {
    config = {
      source      = filemanager_copy.subdir_config.source
      destination = filemanager_copy.subdir_config.destination
    }
    logs = {
      source      = filemanager_copy.subdir_logs.source
      destination = filemanager_copy.subdir_logs.destination
    }
  }
}

# -----------------------------------------------------------------------------
# PERMISSION VARIATIONS
# -----------------------------------------------------------------------------

output "permission_copies" {
  value = {
    restrictive = {
      destination          = filemanager_copy.perms_restrictive.destination
      file_permission      = filemanager_copy.perms_restrictive.file_permission
      directory_permission = filemanager_copy.perms_restrictive.directory_permission
    }
    permissive = {
      destination          = filemanager_copy.perms_permissive.destination
      file_permission      = filemanager_copy.perms_permissive.file_permission
      directory_permission = filemanager_copy.perms_permissive.directory_permission
    }
  }
}

# -----------------------------------------------------------------------------
# DEPLOYMENT/BACKUP
# -----------------------------------------------------------------------------

output "deployment_copies" {
  value = {
    deploy = {
      destination  = filemanager_copy.deploy.destination
      files_copied = filemanager_copy.deploy.files_copied
      excludes     = filemanager_copy.deploy.excludes
    }
    backup = {
      destination          = filemanager_copy.backup.destination
      files_copied         = filemanager_copy.backup.files_copied
      preserve_permissions = filemanager_copy.backup.preserve_permissions
    }
  }
}

# -----------------------------------------------------------------------------
# SPECIAL CASES
# -----------------------------------------------------------------------------

output "special_copies" {
  value = {
    with_spaces = {
      source      = filemanager_copy.special_spaces.source
      destination = filemanager_copy.special_spaces.destination
    }
    unicode = {
      source      = filemanager_copy.special_unicode.source
      destination = filemanager_copy.special_unicode.destination
    }
    dotted = {
      source      = filemanager_copy.special_dots.source
      destination = filemanager_copy.special_dots.destination
    }
  }
}

# -----------------------------------------------------------------------------
# SUMMARY
# -----------------------------------------------------------------------------

output "summary" {
  value = {
    total_copy_operations = 26
    categories = [
      "single_file",
      "directory",
      "overwrite",
      "nested",
      "multi",
      "subdir",
      "permissions",
      "deployment",
      "special"
    ]
  }
}
