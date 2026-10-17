package config

const (
	buildArgPrefix = "build-arg:"
	labelPrefix    = "label:"

	// KeyTarget is the build option selecting the target stage.
	KeyTarget = "target"

	keyCgroupParent     = "cgroup-parent"
	keyForceNetwork     = "force-network-mode"
	keyGlobalAddHosts   = "add-hosts"
	keyHostname         = "hostname"
	keyImageResolveMode = "image-resolve-mode"
	keyMultiPlatform    = "multi-platform"
	keyShmSize          = "shm-size"
	keyTargetPlatform   = "platform"
	keyUlimit           = "ulimit"
	keyCacheImports     = "cache-imports" // JSON of []CacheOptionsEntry

	// Update docs/reference.md when adding a build-arg.
	keyCacheNSArg       = "build-arg:BUILDKIT_CACHE_MOUNT_NS"
	keyMultiPlatformArg = "build-arg:BUILDKIT_MULTI_PLATFORM"
	keyHostnameArg      = "build-arg:BUILDKIT_SANDBOX_HOSTNAME"
	keyFrontendLintArg  = "build-arg:BUILDKIT_FRONTEND_CHECK"
	keySourceDateEpoch  = "build-arg:SOURCE_DATE_EPOCH"
)
