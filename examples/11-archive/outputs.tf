# ARCHIVE RESOURCE - OUTPUTS

# -----------------------------------------------------------------------------
# ZIP ARCHIVES
# -----------------------------------------------------------------------------

output "zip_archives" {
  value = {
    basic = {
      path = filemanager_archive.zip_basic.path
      type = filemanager_archive.zip_basic.type
      size = filemanager_archive.zip_basic.size
    }
    with_excludes = {
      path = filemanager_archive.zip_excludes.path
      type = filemanager_archive.zip_excludes.type
      size = filemanager_archive.zip_excludes.size
    }
    multi_excludes = {
      path = filemanager_archive.zip_multi_excludes.path
      type = filemanager_archive.zip_multi_excludes.type
      size = filemanager_archive.zip_multi_excludes.size
    }
  }
}

# -----------------------------------------------------------------------------
# TAR ARCHIVES
# -----------------------------------------------------------------------------

output "tar_archives" {
  value = {
    basic = {
      path = filemanager_archive.tar_basic.path
      type = filemanager_archive.tar_basic.type
      size = filemanager_archive.tar_basic.size
    }
    with_excludes = {
      path = filemanager_archive.tar_excludes.path
      type = filemanager_archive.tar_excludes.type
      size = filemanager_archive.tar_excludes.size
    }
  }
}

# -----------------------------------------------------------------------------
# TAR.GZ ARCHIVES
# -----------------------------------------------------------------------------

output "targz_archives" {
  value = {
    basic = {
      path = filemanager_archive.targz_basic.path
      type = filemanager_archive.targz_basic.type
      size = filemanager_archive.targz_basic.size
    }
    with_excludes = {
      path = filemanager_archive.targz_excludes.path
      type = filemanager_archive.targz_excludes.type
      size = filemanager_archive.targz_excludes.size
    }
    backup = {
      path = filemanager_archive.backup.path
      type = filemanager_archive.backup.type
      size = filemanager_archive.backup.size
    }
  }
}

# -----------------------------------------------------------------------------
# FORMAT COMPARISON
# -----------------------------------------------------------------------------

output "format_comparison" {
  value = {
    zip = {
      path = filemanager_archive.compare_zip.path
      size = filemanager_archive.compare_zip.size
    }
    tar = {
      path = filemanager_archive.compare_tar.path
      size = filemanager_archive.compare_tar.size
    }
    targz = {
      path = filemanager_archive.compare_targz.path
      size = filemanager_archive.compare_targz.size
    }
  }
  description = "Compare sizes of same content in different formats"
}

# -----------------------------------------------------------------------------
# SPECIAL CASES
# -----------------------------------------------------------------------------

output "special_cases" {
  value = {
    with_empty_dir = {
      path = filemanager_archive.with_empty.path
      size = filemanager_archive.with_empty.size
    }
    nested_structure = {
      path = filemanager_archive.nested.path
      size = filemanager_archive.nested.size
    }
  }
}

# -----------------------------------------------------------------------------
# MULTIPLE ARCHIVES
# -----------------------------------------------------------------------------

output "multi_archives" {
  value = {
    archive1 = {
      path = filemanager_archive.multi_1.path
      size = filemanager_archive.multi_1.size
    }
    archive2 = {
      path = filemanager_archive.multi_2.path
      size = filemanager_archive.multi_2.size
    }
    archive3 = {
      path = filemanager_archive.multi_3.path
      size = filemanager_archive.multi_3.size
    }
  }
}

# -----------------------------------------------------------------------------
# SUMMARY
# -----------------------------------------------------------------------------

output "summary" {
  value = {
    total_archives = 16
    archive_types  = ["zip", "tar", "tar.gz"]
    categories     = ["basic", "excludes", "comparison", "special", "multi"]
  }
}
