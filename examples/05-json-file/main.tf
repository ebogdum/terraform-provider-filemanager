# =============================================================================
# JSON FILE RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/05-json-file"
}

# -----------------------------------------------------------------------------
# BASIC JSON FILES
# -----------------------------------------------------------------------------

# Case 1: Simple key-value
resource "filemanager_json_file" "simple" {
  path = "${local.output_dir}/basic/simple.json"

  content = jsonencode({
    key   = "value"
    count = 42
  })

  create_parent_dirs = true
}

# Case 2: Nested object
resource "filemanager_json_file" "nested" {
  path = "${local.output_dir}/basic/nested.json"

  content = jsonencode({
    level1 = {
      level2 = {
        level3 = {
          deep_value = "found"
        }
      }
    }
  })

  create_parent_dirs = true
}

# Case 3: Array of primitives
resource "filemanager_json_file" "array_primitives" {
  path = "${local.output_dir}/basic/array_primitives.json"

  content = jsonencode([1, 2, 3, 4, 5])

  create_parent_dirs = true
}

# Case 4: Array of objects
resource "filemanager_json_file" "array_objects" {
  path = "${local.output_dir}/basic/array_objects.json"

  content = jsonencode([
    { id = 1, name = "first" },
    { id = 2, name = "second" },
    { id = 3, name = "third" }
  ])

  create_parent_dirs = true
}

# Case 5: Mixed types
resource "filemanager_json_file" "mixed_types" {
  path = "${local.output_dir}/basic/mixed_types.json"

  content = jsonencode({
    string_val = "hello"
    int_val    = 42
    float_val  = 3.14
    bool_true  = true
    bool_false = false
    null_val   = null
    array_val  = [1, "two", true]
    object_val = { nested = "value" }
  })

  create_parent_dirs = true
}

# Case 6: Empty object
resource "filemanager_json_file" "empty_object" {
  path = "${local.output_dir}/basic/empty_object.json"

  content = jsonencode({})

  create_parent_dirs = true
}

# Case 7: Empty array
resource "filemanager_json_file" "empty_array" {
  path = "${local.output_dir}/basic/empty_array.json"

  content = jsonencode([])

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# MERGE STRATEGIES
# -----------------------------------------------------------------------------

# Case 8: Deep merge - nested objects combined
resource "filemanager_json_file" "merge_deep" {
  path = "${local.output_dir}/merge/deep.json"

  content = jsonencode({
    database = {
      host = "localhost"
      port = 5432
    }
    logging = {
      level = "info"
    }
  })

  merge_with = jsonencode({
    database = {
      name     = "mydb"
      username = "admin"
    }
    cache = {
      enabled = true
      ttl     = 3600
    }
  })

  merge_strategy     = "deep"
  create_parent_dirs = true
}

# Case 9: Replace merge - overlay replaces base
resource "filemanager_json_file" "merge_replace" {
  path = "${local.output_dir}/merge/replace.json"

  content = jsonencode({
    original_key = "original_value"
    shared_key   = "from_base"
  })

  merge_with = jsonencode({
    shared_key = "from_overlay"
    new_key    = "new_value"
  })

  merge_strategy     = "replace"
  create_parent_dirs = true
}

# Case 10: Append merge - arrays concatenated
resource "filemanager_json_file" "merge_append" {
  path = "${local.output_dir}/merge/append.json"

  content = jsonencode({
    items = [1, 2, 3]
  })

  merge_with = jsonencode({
    items = [4, 5, 6]
  })

  merge_strategy     = "append"
  create_parent_dirs = true
}

# Case 11: Complex deep merge
resource "filemanager_json_file" "merge_complex" {
  path = "${local.output_dir}/merge/complex.json"

  content = jsonencode({
    app = {
      name    = "MyApp"
      version = ">= 1.0.0"
      config = {
        debug    = false
        features = ["feature1", "feature2"]
      }
    }
    server = {
      host = "0.0.0.0"
    }
  })

  merge_with = jsonencode({
    app = {
      environment = "production"
      config = {
        timeout  = 30
        features = ["feature3"]
      }
    }
    server = {
      port = 8080
      ssl  = true
    }
    database = {
      host = "db.example.com"
    }
  })

  merge_strategy     = "deep"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# FORMATTING OPTIONS
# -----------------------------------------------------------------------------

# Case 12: With sort_keys
resource "filemanager_json_file" "sorted" {
  path = "${local.output_dir}/formatting/sorted.json"

  content = jsonencode({
    zebra  = "last"
    alpha  = "first"
    middle = "mid"
  })

  sort_keys          = true
  create_parent_dirs = true
}

# Case 13: Without sort_keys (original order)
resource "filemanager_json_file" "unsorted" {
  path = "${local.output_dir}/formatting/unsorted.json"

  content = jsonencode({
    zebra  = "last"
    alpha  = "first"
    middle = "mid"
  })

  sort_keys          = false
  create_parent_dirs = true
}

# Case 14: Custom indent (4 spaces)
resource "filemanager_json_file" "indent_4" {
  path = "${local.output_dir}/formatting/indent_4.json"

  content = jsonencode({
    key = {
      nested = "value"
    }
  })

  indent             = 4
  create_parent_dirs = true
}

# Case 15: No indent (1 space)
resource "filemanager_json_file" "indent_1" {
  path = "${local.output_dir}/formatting/indent_1.json"

  content = jsonencode({
    key = {
      nested = "value"
    }
  })

  indent             = 1
  create_parent_dirs = true
}

# Case 16: Compact (minimal whitespace)
resource "filemanager_json_file" "compact" {
  path = "${local.output_dir}/formatting/compact.json"

  content = jsonencode({
    key    = "value"
    nested = { a = 1, b = 2 }
  })

  compact            = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# REAL-WORLD CONFIGURATIONS
# -----------------------------------------------------------------------------

# Case 17: Package.json
resource "filemanager_json_file" "package_json" {
  path = "${local.output_dir}/configs/package.json"

  content = jsonencode({
    name        = "my-app"
    version     = "1.0.0"
    description = "My awesome application"
    main        = "index.js"
    scripts = {
      start = "node index.js"
      test  = "jest"
      build = "webpack --mode production"
    }
    dependencies = {
      express = "^4.18.0"
      lodash  = "^4.17.0"
    }
    devDependencies = {
      jest    = "^29.0.0"
      webpack = "^5.0.0"
    }
    keywords = ["app", "node", "express"]
    author   = "Developer"
    license  = "MIT"
  })

  sort_keys          = true
  indent             = 2
  create_parent_dirs = true
}

# Case 18: tsconfig.json
resource "filemanager_json_file" "tsconfig" {
  path = "${local.output_dir}/configs/tsconfig.json"

  content = jsonencode({
    compilerOptions = {
      target          = "ES2020"
      module          = "commonjs"
      lib             = ["ES2020"]
      strict          = true
      esModuleInterop = true
      skipLibCheck    = true
      outDir          = "./dist"
      rootDir         = "./src"
      declaration     = true
      sourceMap       = true
    }
    include = ["src/**/*"]
    exclude = ["node_modules", "dist"]
  })

  indent             = 2
  create_parent_dirs = true
}

# Case 19: API response mock
resource "filemanager_json_file" "api_response" {
  path = "${local.output_dir}/configs/api_response.json"

  content = jsonencode({
    status = 200
    data = {
      users = [
        {
          id       = 1
          username = "john_doe"
          email    = "john@example.com"
          roles    = ["admin", "user"]
          profile = {
            firstName = "John"
            lastName  = "Doe"
            avatar    = "https://example.com/avatars/1.jpg"
          }
        },
        {
          id       = 2
          username = "jane_doe"
          email    = "jane@example.com"
          roles    = ["user"]
          profile = {
            firstName = "Jane"
            lastName  = "Doe"
            avatar    = null
          }
        }
      ]
      pagination = {
        page       = 1
        perPage    = 10
        total      = 2
        totalPages = 1
      }
    }
    meta = {
      timestamp = "2024-01-15T10:30:00Z"
      version   = "1.0"
    }
  })

  indent             = 2
  create_parent_dirs = true
}

# Case 20: Docker compose override
resource "filemanager_json_file" "vscode_settings" {
  path = "${local.output_dir}/configs/settings.json"

  content = jsonencode({
    "editor.fontSize"              = 14
    "editor.tabSize"               = 2
    "editor.formatOnSave"          = true
    "files.autoSave"               = "afterDelay"
    "files.autoSaveDelay"          = 1000
    "terminal.integrated.fontSize" = 13
    "workbench.colorTheme"         = "One Dark Pro"
  })

  indent             = 2
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Case 21: Unicode content
resource "filemanager_json_file" "unicode" {
  path = "${local.output_dir}/edge/unicode.json"

  content = jsonencode({
    japanese = "日本語"
    chinese  = "中文"
    korean   = "한국어"
    arabic   = "العربية"
    emoji    = "🎉🚀💻"
    special  = "quotes: \"hello\" and 'world'"
  })

  create_parent_dirs = true
}

# Case 22: Large numbers
resource "filemanager_json_file" "numbers" {
  path = "${local.output_dir}/edge/numbers.json"

  content = jsonencode({
    integer    = 123456789
    float      = 3.141592653589793
    negative   = -9999
    zero       = 0
    scientific = 1.23e10
  })

  create_parent_dirs = true
}

# Case 23: Deeply nested structure
resource "filemanager_json_file" "deeply_nested" {
  path = "${local.output_dir}/edge/deeply_nested.json"

  content = jsonencode({
    l1 = {
      l2 = {
        l3 = {
          l4 = {
            l5 = {
              l6 = {
                l7 = {
                  l8 = {
                    value = "very deep"
                  }
                }
              }
            }
          }
        }
      }
    }
  })

  indent             = 2
  create_parent_dirs = true
}

# Case 24: Large array
resource "filemanager_json_file" "large_array" {
  path = "${local.output_dir}/edge/large_array.json"

  content = jsonencode([for i in range(100) : { id = i, name = "item-${i}" }])

  indent             = 2
  create_parent_dirs = true
}

# Case 25: String with special JSON characters
resource "filemanager_json_file" "special_chars" {
  path = "${local.output_dir}/edge/special_chars.json"

  content = jsonencode({
    backslash   = "path\\to\\file"
    newline     = "line1\nline2"
    tab         = "col1\tcol2"
    quote       = "he said \"hello\""
    unicode_esc = "symbol: \\u0041"
  })

  create_parent_dirs = true
}
