name: "hof cli"

image: "mcr.microsoft.com/devcontainers/universal:2"

postCreateCommand: """
sudo rm -rf /usr/local/hugo
"""

customizations: {
	vscode: extensions: [
		"asdine.cue",
		"jallen7usa.vscode-cue-fmt",
	]
}
