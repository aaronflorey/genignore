package customtemplate

type Definition struct {
	Key  string
	Path string
}

// Definitions maps custom provider keys to embedded template files.
//
// To add another embedded custom template:
// 1) Add a new .gitignore file under internal/customtemplate/templates/.
// 2) Register it here with a unique key and relative file path.
var Definitions = []Definition{
	{Key: "ai-agents", Path: "templates/ai-agents.gitignore"},
}
