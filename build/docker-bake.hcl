group "default" {
  targets = [
    "1_0_0",
  ]
}

target "build-dockerfile" {
  dockerfile = "Dockerfile"
}

target "build-platforms" {
  platforms = ["linux/amd64", "linux/aarch64"]
}

target "build-common" {
  pull = true
}

variable "REGISTRY_CACHE" {
  default = "ghcr.io/xZero707/telegram-event-server-cache"
}

######################
# Define the functions
######################

# Get the arguments for the build
function "get-args" {
  params = [version, patch_version]
  result = {
    WP_VERSION = version
    WP_PATCH_VERSION = patch_version
  }
}

# Get the cache-from configuration
function "get-cache-from" {
  params = [version]
  result = [
    "type=registry,ref=${REGISTRY_CACHE}:${sha1("${version}-${BAKE_LOCAL_PLATFORM}")}"
  ]
}

# Get the cache-to configuration
function "get-cache-to" {
  params = [version]
  result = [
    "type=registry,mode=max,ref=${REGISTRY_CACHE}:${sha1("${version}-${BAKE_LOCAL_PLATFORM}")}"
  ]
}

# Get list of image tags and registries
# Takes a version and a list of extra versions to tag
# eg. get-tags("1.0.0", ["1", "1.0", "latest"])
function "get-tags" {
  params = [version, extra_versions]
  result = concat(
    [
      "ghcr.io/xZero707/telegram-event-server:${version}"
    ],
    flatten([
      for extra_version in extra_versions : [
        "ghcr.io/xZero707/telegram-event-server:${extra_version}"
      ]
    ])
  )
}

##########################
# Define the build targets
##########################

target "1_0_0" {
  inherits   = ["build-dockerfile", "build-platforms", "build-common"]
  cache-from = get-cache-from("1.0.0")
  cache-to   = get-cache-to("1.0.0")
  tags       = get-tags("1.0.0", ["1", "1.0", "latest"])
  args       = get-args("1.0.0", "1.0.0")
}
