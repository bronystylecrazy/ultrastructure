package us

var (
	Name        string = "Ultrastructure"
	Description string = "a lightweight web framework for Go based on UberFX."
)

func IsProduction() bool {
	return Version != "v0.0.0-development"
}

func IsDevelopment() bool {
	return Version == "v0.0.0-development"
}
