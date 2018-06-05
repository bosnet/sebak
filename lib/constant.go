package sebak

const (
	// Version is Top-level of version. It must follow SemVer (https://semver.org)
	Version = "0.1.0+proto"

	// BaseFee is the default transaction fee, if fee is lower than BaseFee, the
	// transaction will fail validation.
	BaseFee Amount = 10000
)
