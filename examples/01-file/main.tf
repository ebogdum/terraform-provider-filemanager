# =============================================================================
# FILE RESOURCE - ALL USE CASES
# =============================================================================

terraform {
  required_providers {
    filemanager = {
      source  = "ebogdum/filemanager"
      version = ">= 1.0.0"
    }
  }
}

provider "filemanager" {}

locals {
  output_dir = "${path.module}/../../test/output/01-file"
}

# -----------------------------------------------------------------------------
# BASIC FILE OPERATIONS
# -----------------------------------------------------------------------------

# Case 1: Basic file with simple content
resource "filemanager_file" "basic" {
  path    = "${local.output_dir}/basic.txt"
  content = "Hello, World!"

  create_parent_dirs = true
}

# Case 2: Empty file
resource "filemanager_file" "empty" {
  path    = "${local.output_dir}/empty.txt"
  content = ""

  create_parent_dirs = true
}

# Case 3: Multi-line content
resource "filemanager_file" "multiline" {
  path    = "${local.output_dir}/multiline.txt"
  content = <<-EOF
    Line 1
    Line 2
    Line 3

    Line 5 (after empty line)
  EOF

  create_parent_dirs = true
}

# Case 4: Large content (stress test)
resource "filemanager_file" "large" {
  path    = "${local.output_dir}/large.txt"
  content = join("\n", [for i in range(1000) : "Line ${i}: ${uuid()}"])

  create_parent_dirs = true
}

# Case 5: Special characters
resource "filemanager_file" "special_chars" {
  path    = "${local.output_dir}/special_chars.txt"
  content = <<-EOF
    Special characters: !@#$%^&*()_+-=[]{}|;':",.<>?/\`~
    Unicode: 日本語 中文 한국어 العربية עברית
    Emoji: 🎉 🚀 ✨ 💻 🔥
    Escape sequences: \n \t \r \\
  EOF

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# PERMISSION VARIATIONS
# -----------------------------------------------------------------------------

# Case 6: Read-only file (0444)
resource "filemanager_file" "readonly" {
  path            = "${local.output_dir}/permissions/readonly.txt"
  content         = "This file is read-only"
  file_permission = "0444"

  create_parent_dirs = true
}

# Case 7: Owner read-write only (0600)
resource "filemanager_file" "owner_only" {
  path            = "${local.output_dir}/permissions/owner_only.txt"
  content         = "Only owner can read/write"
  file_permission = "0600"

  create_parent_dirs = true
}

# Case 8: Full permissions (0777)
resource "filemanager_file" "full_perms" {
  path            = "${local.output_dir}/permissions/full_perms.txt"
  content         = "Everyone can do everything"
  file_permission = "0777"

  create_parent_dirs = true
}

# Case 9: Executable script (0755)
resource "filemanager_file" "executable" {
  path            = "${local.output_dir}/permissions/script.sh"
  content         = "#!/bin/bash\necho 'Hello from script'"
  file_permission = "0755"

  create_parent_dirs = true
}

# Case 10: Custom directory permissions
resource "filemanager_file" "custom_dir_perms" {
  path                 = "${local.output_dir}/custom_perms/file.txt"
  content              = "File in custom permission directory"
  file_permission      = "0644"
  directory_permission = "0700"

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# ENCODING VARIATIONS
# -----------------------------------------------------------------------------

# Case 11: Base64 encoded content
resource "filemanager_file" "base64_content" {
  path           = "${local.output_dir}/encoding/from_base64.txt"
  content_base64 = base64encode("This was base64 encoded")

  create_parent_dirs = true
}

# Case 12: Binary-like content via base64
resource "filemanager_file" "binary_via_base64" {
  path = "${local.output_dir}/encoding/binary.bin"
  # Pre-encoded binary content (PNG header bytes)
  content_base64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# ATOMIC WRITE & CHECKSUM VERIFICATION
# -----------------------------------------------------------------------------

# Case 13: With atomic write enabled
resource "filemanager_file" "atomic" {
  path         = "${local.output_dir}/atomic/atomic_write.txt"
  content      = "This file was written atomically"
  atomic_write = true

  create_parent_dirs = true
}

# Case 14: With checksum verification
resource "filemanager_file" "verified" {
  path            = "${local.output_dir}/atomic/verified.txt"
  content         = "This file has verified checksum"
  verify_checksum = true

  create_parent_dirs = true
}

# Case 15: Both atomic and verified
resource "filemanager_file" "atomic_verified" {
  path            = "${local.output_dir}/atomic/atomic_verified.txt"
  content         = "Atomic write with checksum verification"
  atomic_write    = true
  verify_checksum = true

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# NEWLINE HANDLING
# -----------------------------------------------------------------------------

# Case 16: Unix newlines (LF)
resource "filemanager_file" "newline_lf" {
  path    = "${local.output_dir}/newlines/unix.txt"
  content = "Line 1\nLine 2\nLine 3"
  newline = "lf"

  create_parent_dirs = true
}

# Case 17: Windows newlines (CRLF)
resource "filemanager_file" "newline_crlf" {
  path    = "${local.output_dir}/newlines/windows.txt"
  content = "Line 1\nLine 2\nLine 3"
  newline = "crlf"

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# FORCE OVERWRITE
# -----------------------------------------------------------------------------

# Case 18: With force enabled
resource "filemanager_file" "with_force" {
  path    = "${local.output_dir}/force/forced.txt"
  content = "This file can be overwritten"
  force   = true

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# NESTED DIRECTORIES
# -----------------------------------------------------------------------------

# Case 19: Deeply nested path
resource "filemanager_file" "deeply_nested" {
  path    = "${local.output_dir}/level1/level2/level3/level4/level5/deep.txt"
  content = "Deep in the directory tree"

  create_parent_dirs = true
}

# Case 20: Multiple files in same nested structure
resource "filemanager_file" "nested_file1" {
  path    = "${local.output_dir}/nested/shared/file1.txt"
  content = "First file in shared directory"

  create_parent_dirs = true
}

resource "filemanager_file" "nested_file2" {
  path    = "${local.output_dir}/nested/shared/file2.txt"
  content = "Second file in shared directory"

  create_parent_dirs = true
}

resource "filemanager_file" "nested_file3" {
  path    = "${local.output_dir}/nested/shared/subdir/file3.txt"
  content = "Third file in subdirectory"

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# FILE EXTENSIONS
# -----------------------------------------------------------------------------

# Case 21: Various file extensions
resource "filemanager_file" "ext_json" {
  path               = "${local.output_dir}/extensions/config.json"
  content            = "{\"key\": \"value\"}"
  create_parent_dirs = true
}

resource "filemanager_file" "ext_yaml" {
  path               = "${local.output_dir}/extensions/config.yaml"
  content            = "key: value"
  create_parent_dirs = true
}

resource "filemanager_file" "ext_xml" {
  path               = "${local.output_dir}/extensions/config.xml"
  content            = "<?xml version=\"1.0\"?><root><key>value</key></root>"
  create_parent_dirs = true
}

resource "filemanager_file" "ext_html" {
  path               = "${local.output_dir}/extensions/index.html"
  content            = "<!DOCTYPE html><html><body>Hello</body></html>"
  create_parent_dirs = true
}

resource "filemanager_file" "ext_css" {
  path               = "${local.output_dir}/extensions/style.css"
  content            = "body { color: black; }"
  create_parent_dirs = true
}

resource "filemanager_file" "ext_js" {
  path               = "${local.output_dir}/extensions/script.js"
  content            = "console.log('hello');"
  create_parent_dirs = true
}

resource "filemanager_file" "ext_md" {
  path               = "${local.output_dir}/extensions/readme.md"
  content            = "# Title\n\nContent here"
  create_parent_dirs = true
}

resource "filemanager_file" "no_extension" {
  path               = "${local.output_dir}/extensions/Makefile"
  content            = "all:\n\techo 'building'"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Case 22: File with spaces in name
resource "filemanager_file" "spaces_in_name" {
  path               = "${local.output_dir}/edge-cases/file with spaces.txt"
  content            = "Filename has spaces"
  create_parent_dirs = true
}

# Case 23: File with dots in name
resource "filemanager_file" "dots_in_name" {
  path               = "${local.output_dir}/edge-cases/file.multiple.dots.txt"
  content            = "Filename has multiple dots"
  create_parent_dirs = true
}

# Case 24: Hidden file (starts with dot)
resource "filemanager_file" "hidden_file" {
  path               = "${local.output_dir}/edge-cases/.hidden"
  content            = "This is a hidden file"
  create_parent_dirs = true
}

# Case 25: Very long filename
resource "filemanager_file" "long_filename" {
  path               = "${local.output_dir}/edge-cases/this_is_a_very_long_filename_that_tests_the_limits_of_the_filesystem.txt"
  content            = "Long filename test"
  create_parent_dirs = true
}

# Case 26: Only whitespace content
resource "filemanager_file" "whitespace_only" {
  path               = "${local.output_dir}/edge-cases/whitespace.txt"
  content            = "   \n\t\n   "
  create_parent_dirs = true
}

# Case 27: Single character content
resource "filemanager_file" "single_char" {
  path               = "${local.output_dir}/edge-cases/single.txt"
  content            = "X"
  create_parent_dirs = true
}

# Case 28: Newline only content
resource "filemanager_file" "newline_only" {
  path               = "${local.output_dir}/edge-cases/newline.txt"
  content            = "\n"
  create_parent_dirs = true
}
