# =============================================================================
# VERIFICATION CHECKS - Archive resource validation
# =============================================================================

check "verify_zip_basic" {
  data "filemanager_stat" "zip_basic_check" {
    path = filemanager_archive.zip_basic.path
  }

  assert {
    condition     = data.filemanager_stat.zip_basic_check.exists == true
    error_message = "Basic ZIP archive should exist"
  }

  assert {
    condition     = data.filemanager_stat.zip_basic_check.size > 0
    error_message = "Basic ZIP archive should not be empty"
  }
}

check "verify_zip_excludes" {
  data "filemanager_stat" "zip_excludes_check" {
    path = filemanager_archive.zip_excludes.path
  }

  assert {
    condition     = data.filemanager_stat.zip_excludes_check.exists == true
    error_message = "ZIP archive with excludes should exist"
  }
}

check "verify_tar_basic" {
  data "filemanager_stat" "tar_basic_check" {
    path = filemanager_archive.tar_basic.path
  }

  assert {
    condition     = data.filemanager_stat.tar_basic_check.exists == true
    error_message = "Basic TAR archive should exist"
  }

  assert {
    condition     = data.filemanager_stat.tar_basic_check.size > 0
    error_message = "Basic TAR archive should not be empty"
  }
}

check "verify_targz_basic" {
  data "filemanager_stat" "targz_basic_check" {
    path = filemanager_archive.targz_basic.path
  }

  assert {
    condition     = data.filemanager_stat.targz_basic_check.exists == true
    error_message = "tar.gz archive should exist"
  }
}

check "verify_targz_excludes" {
  data "filemanager_stat" "targz_excludes_check" {
    path = filemanager_archive.targz_excludes.path
  }

  assert {
    condition     = data.filemanager_stat.targz_excludes_check.exists == true
    error_message = "tar.gz archive with excludes should exist"
  }
}

check "verify_backup_archive" {
  data "filemanager_stat" "backup_check" {
    path = filemanager_archive.backup.path
  }

  assert {
    condition     = data.filemanager_stat.backup_check.exists == true
    error_message = "Backup archive should exist"
  }
}

check "verify_archive_checksum" {
  data "filemanager_checksum" "archive_checksum" {
    path      = filemanager_archive.zip_basic.path
    algorithm = "sha256"
  }

  assert {
    condition     = data.filemanager_checksum.archive_checksum.checksum != ""
    error_message = "Archive should have a valid SHA256 checksum"
  }
}

# =============================================================================
# ARCHIVE TYPE COMPARISON CHECKS
# =============================================================================

check "verify_compare_zip" {
  data "filemanager_stat" "compare_zip" {
    path = filemanager_archive.compare_zip.path
  }

  assert {
    condition     = data.filemanager_stat.compare_zip.exists == true
    error_message = "Comparison ZIP archive should exist"
  }
}

check "verify_compare_tar" {
  data "filemanager_stat" "compare_tar" {
    path = filemanager_archive.compare_tar.path
  }

  assert {
    condition     = data.filemanager_stat.compare_tar.exists == true
    error_message = "Comparison TAR archive should exist"
  }
}

check "verify_compare_targz" {
  data "filemanager_stat" "compare_targz" {
    path = filemanager_archive.compare_targz.path
  }

  assert {
    condition     = data.filemanager_stat.compare_targz.exists == true
    error_message = "Comparison tar.gz archive should exist"
  }
}

# =============================================================================
# ENHANCED STAT CHECKS (time-based)
# =============================================================================

check "stat_archive_time_check" {
  data "filemanager_stat" "archive_time_check" {
    path            = filemanager_archive.zip_basic.path
    modified_within = "1h"
  }

  assert {
    condition     = data.filemanager_stat.archive_time_check.is_modified_within == true
    error_message = "Newly created archive should be modified within last hour"
  }
}
