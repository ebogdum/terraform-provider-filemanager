# =============================================================================
# XML FILE RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/17-xml-file"
}

# -----------------------------------------------------------------------------
# BASIC XML FILES
# -----------------------------------------------------------------------------

# Case 1: Simple XML document
resource "filemanager_xml_file" "simple" {
  path = "${local.output_dir}/basic/simple.xml"

  content = jsonencode({
    root = {
      name    = "example"
      version = "1.0"
    }
  })

  create_parent_dirs = true
}

# Case 2: XML with attributes
resource "filemanager_xml_file" "with_attributes" {
  path = "${local.output_dir}/basic/attributes.xml"

  content = jsonencode({
    book = {
      "@id"   = "123"
      "@isbn" = "978-0-123456-78-9"
      title   = "Terraform Guide"
      author  = "John Doe"
      year    = "2024"
    }
  })

  create_parent_dirs = true
}

# Case 3: Nested XML structure
resource "filemanager_xml_file" "nested" {
  path = "${local.output_dir}/basic/nested.xml"

  content = jsonencode({
    catalog = {
      books = {
        book = [
          {
            "@id"  = "1"
            title  = "Book One"
            author = "Author A"
          },
          {
            "@id"  = "2"
            title  = "Book Two"
            author = "Author B"
          }
        ]
      }
    }
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# CONFIG FILES
# -----------------------------------------------------------------------------

# Case 4: Maven POM-like file
resource "filemanager_xml_file" "pom" {
  path = "${local.output_dir}/configs/pom.xml"

  content = jsonencode({
    project = {
      "@xmlns"     = "http://maven.apache.org/POM/4.0.0"
      modelVersion = "4.0.0"
      groupId      = "com.example"
      artifactId   = "my-app"
      version      = "1.0.0"
      packaging    = "jar"
      dependencies = {
        dependency = [
          {
            groupId    = "org.junit"
            artifactId = "junit"
            version    = "5.0.0"
            scope      = "test"
          }
        ]
      }
    }
  })
  indent = 4

  create_parent_dirs = true
}

# Case 5: Spring beans config
resource "filemanager_xml_file" "spring_beans" {
  path = "${local.output_dir}/configs/beans.xml"

  content = jsonencode({
    beans = {
      "@xmlns" = "http://www.springframework.org/schema/beans"
      bean = [
        {
          "@id"    = "dataSource"
          "@class" = "com.example.DataSource"
          property = {
            "@name"  = "url"
            "@value" = "jdbc:mysql://localhost/db"
          }
        }
      ]
    }
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# FORMATTING OPTIONS
# -----------------------------------------------------------------------------

# Case 6: Compact XML (no indentation)
resource "filemanager_xml_file" "compact" {
  path = "${local.output_dir}/formatting/compact.xml"

  content = jsonencode({
    data = {
      item = "value"
    }
  })

  compact = true

  create_parent_dirs = true
}

# Case 7: Custom indentation
resource "filemanager_xml_file" "custom_indent" {
  path = "${local.output_dir}/formatting/indent_4.xml"

  content = jsonencode({
    config = {
      setting = {
        name  = "debug"
        value = "true"
      }
    }
  })

  indent = 4

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Case 8: Empty root element
resource "filemanager_xml_file" "empty_root" {
  path = "${local.output_dir}/edge/empty.xml"

  content = jsonencode({
    empty = {}
  })

  create_parent_dirs = true
}

# Case 9: Special characters in content
resource "filemanager_xml_file" "special_chars" {
  path = "${local.output_dir}/edge/special.xml"

  content = jsonencode({
    data = {
      text    = "Special chars: <>&'\""
      unicode = "Unicode: "
      cdata   = "Some CDATA content"
    }
  })

  create_parent_dirs = true
}

