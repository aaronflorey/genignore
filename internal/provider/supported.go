package provider

var SupportedKeys = []string{
	"android", "androidstudio", "angular", "ansible", "ansibletower", "appcode", "appcode+all", "appcode+iml", "archive", "archives", "archlinuxpackages", "asdf", "aspnetcore", "astro", "audio", "azurefunctions", "azurite", "backup", "basic", "blender", "bower", "c", "c++", "cake", "certificates", "chocolatey", "clean", "clion", "clion+all", "clion+iml", "clojure", "cmake", "cocoapods", "codeigniter", "codeio", "codesniffer", "composer", "compressed", "compressedarchive", "compression", "cordova", "crystal", "csharp", "cvs", "dart", "data", "database", "dbeaver", "deno", "diff", "direnv", "diskimage", "django", "docz", "dotenv", "dotfilessh", "dotnetcore", "dotsettings", "dreamweaver", "dropbox", "drupal", "drupal7", "drupal8", "eclipse", "elasticbeanstalk", "elixir", "emacs", "erlang", "executable", "fastlane", "firebase", "fish", "flask", "flatpak", "flutter", "font", "games", "gatsby", "gis", "git", "gitbook", "go", "godot", "goland", "goland+all", "goland+iml", "gpg", "gradle", "groovy", "grunt", "haskell", "helm", "homebrew", "images", "intellij", "intellij+all", "intellij+iml", "java", "jekyll", "jetbrains", "jetbrains+all", "jetbrains+iml", "joomla", "justcode", "kotlin", "lamp", "laravel", "latex", "less", "linux", "localstack", "lua", "macos", "maven", "nanoc", "nextjs", "nim", "node", "now", "nuxtjs", "nwjs", "objective-c", "opencv", "php-cs-fixer", "phpcodesniffer", "phpstorm", "phpstorm+all", "phpstorm+iml", "phpunit", "powershell", "putty", "pycharm", "pycharm+all", "pycharm+iml", "pydev", "python", "rails", "react", "replit", "rider", "ruby", "rubymine", "rubymine+all", "rubymine+iml", "rust", "rust-analyzer", "sass", "serverless", "snyk", "spreadsheet", "ssh", "sublimetext", "svelte", "svn", "swift", "swiftpackagemanager", "symfony", "terraform", "terragrunt", "test", "text", "venv", "vercel", "video", "vim", "virtualenv", "visualstudio", "visualstudiocode", "vue", "vuejs", "waf", "web", "webstorm", "webstorm+all", "webstorm+iml", "windows", "wordpress", "xcode", "yarn", "zig", "zsh",
}

var supportedSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(SupportedKeys))
	for _, key := range SupportedKeys {
		m[key] = struct{}{}
	}
	return m
}()

func IsSupported(key string) bool {
	_, ok := supportedSet[key]
	return ok
}
