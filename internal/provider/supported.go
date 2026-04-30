package provider

import (
	"fmt"
	"slices"

	"github.com/aaronflorey/genignore/internal/customtemplate"
)

var remoteSupportedKeys = []string{
	"actionscript", "ada", "adventuregamestudio", "agda", "al", "android", "angular", "anjuta", "ansible", "appceleratortitanium", "appengine", "archives", "archlinuxpackages", "autotools", "backup", "ballerina", "bazaar", "bricxcc", "bun", "c", "c++", "cakephp", "calabash", "cfwheels", "chefcookbook", "clojure", "cloud9", "cmake", "codeigniter", "codekit", "commonlisp", "composer", "concrete5", "coq", "craftcms", "cuda", "cursor", "cvs", "d", "dart", "darteditor", "delphi", "deno", "diff", "dm", "dotnet", "dreamweaver", "dropbox", "drupal", "eagle", "eclipse", "ecu.test", "eiffelstudio", "elisp", "elixir", "elm", "emacs", "ensime", "episerver", "erlang", "espresso", "expressionengine", "extjs", "fancy", "finale", "firebase", "flaxengine", "flexbuilder", "flutter", "forcedotcom", "fortran", "fuelphp", "gcov", "gitbook", "githubpages", "gleam", "go", "godot", "gpg", "gradle", "grails", "gwt", "haskell", "haxe", "hip", "iar", "idris", "igorpro", "images", "java", "jboss", "jdeveloper", "jekyll", "jenkins_home", "jenv", "jetbrains", "joomla", "julia", "katalon", "kate", "kdevelop4", "kicad", "kohana", "kotlin", "labview", "langchain", "laravel", "lazarus", "lean", "lefthook", "leiningen", "lemonstand", "libreoffice", "lilypond", "linux", "lithium", "lua", "luau", "lyx", "macos", "magento", "matlab", "maven", "mercurial", "mercury", "metals", "metaprogrammingsystem", "microsoftoffice", "mise", "modelica", "modelsim", "momentics", "monodevelop", "moonbit", "nanoc", "nestjs", "netbeans", "nextjs", "nim", "ninja", "nix", "node", "notepadpp", "objective-c", "ocaml", "octave", "ohmyopenagent", "opa", "opencart", "oracleforms", "otto", "packer", "patch", "perl", "phalcon", "platformio", "playframework", "plone", "prestashop", "processing", "psoccreator", "purescript", "putty", "python", "qooxdoo", "qt", "r", "racket", "rails", "raku", "redcar", "redis", "rescript", "rhodesrhomobile", "ros", "ruby", "rust", "salesforce", "sass", "sbt", "scala", "scheme", "scons", "scrivener", "sdcc", "seamgen", "sketchup", "slickedit", "smalltalk", "solidity-remix", "solidworks", "ssdt-sqlproj", "stata", "stella", "stm32cubeide", "sublimetext", "sugarcrm", "svn", "swift", "symfony", "symphonycms", "syncthing", "synopsysvcs", "tags", "terraform", "testcomplete", "tex", "textmate", "textpattern", "tortoisegit", "turbogears2", "twincat3", "typo3", "unity", "unrealengine", "vagrant", "vba", "vim", "virtualenv", "virtuoso", "visualstudio", "visualstudiocode", "vvvv", "waf", "webmethods", "windows", "wordpress", "xcode", "xilinxise", "xojo", "yeoman", "yii", "zed", "zendframework", "zephir", "zig",
}

var (
	SupportedKeys []string
	supportedSet  map[string]struct{}
	initErr       error
)

func init() {
	SupportedKeys = []string{}
	supportedSet = map[string]struct{}{}

	if err := customtemplate.InitError(); err != nil {
		initErr = fmt.Errorf("load embedded custom templates: %w", err)
		return
	}

	remoteSet := make(map[string]struct{}, len(remoteSupportedKeys))
	for _, key := range remoteSupportedKeys {
		remoteSet[key] = struct{}{}
	}

	SupportedKeys = append([]string(nil), remoteSupportedKeys...)
	for _, key := range customtemplate.ProviderKeys() {
		if _, exists := remoteSet[key]; exists {
			initErr = fmt.Errorf("embedded custom template key collides with remote provider key: %s", key)
			SupportedKeys = []string{}
			supportedSet = map[string]struct{}{}
			return
		}
		SupportedKeys = append(SupportedKeys, key)
	}
	slices.Sort(SupportedKeys)
	SupportedKeys = slices.Compact(SupportedKeys)

	supportedSet = make(map[string]struct{}, len(SupportedKeys))
	for _, key := range SupportedKeys {
		supportedSet[key] = struct{}{}
	}
}

func InitError() error {
	return initErr
}

func RemoteSupportedKeys() []string {
	return append([]string(nil), remoteSupportedKeys...)
}

func IsSupported(key string) bool {
	_, ok := supportedSet[key]
	return ok
}
