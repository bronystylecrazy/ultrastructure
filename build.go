package us

func IsProduction() bool {
	return Version != "v0.0.0-development"
}

func IsDevelopment() bool {
	return Version == "v0.0.0-development"
}
