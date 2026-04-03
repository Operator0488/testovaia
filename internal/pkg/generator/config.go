package generator

type Config struct {
	SpecPath    string
	GenPackage  string
	HandlerDir  string
	ServiceRoot string

	APIDir string
	Check  bool
}
