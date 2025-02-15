package common

import "strings"

Steps: {

	checkout: {
		name: "Checkout code"
		uses: "actions/checkout@v3"
	}

	vars: {
		name: "Setup Vars"
		id:   "vars"
		run: """
			SHA=${GITHUB_SHA::8}
			TAG=$(git tag --points-at HEAD)
			echo "HOF_FMT_VERSION=${TAG}" >> $GITHUB_ENV
			if [ -z $TAG ]; then
				TAG=${SHA}
			fi
			echo "HOF_SHA=${SHA}" >> $GITHUB_ENV
			echo "HOF_TAG=${TAG}" >> $GITHUB_ENV
			"""
	}

	cue: {
		install: {
			#ver: string | *"v0.5.0-beta.5"
			run:  """
			mkdir tmp
			cd tmp
			wget https://github.com/cue-lang/cue/releases/download/\(#ver)/cue_\(#ver)_linux_amd64.tar.gz -O cue.tar.gz
			tar -xf cue.tar.gz
			sudo mv cue /usr/local/bin/cue
			cd ../
			rm -rf tmp
			"""
		}
	}

	go: {
		setup: {
			#ver: string | *(string & Versions.go)
			name: "Install Go"
			uses: "actions/setup-go@v3"
			with: "go-version": #ver
		}
		cache: {
			uses: "actions/cache@v3"
			with: {
				path: #"""
					~/go/pkg/mod
					~/.cache/go-build
					~/Library/Caches/go-build
					~\AppData\Local\go-build
					"""#
				key:            "${{ runner.os }}-go-${{ matrix.go-version }}-${{ hashFiles('**/go.sum') }}"
				"restore-keys": "${{ runner.os }}-go-${{ matrix.go-version }}-"
			}
		}
		deps: {
			name: "Fetch Go deps"
			run:  "go mod download"
		}
		releaser: {
			name: "Run GoReleaser"
			uses: "goreleaser/goreleaser-action@v4"
			with: {
				// either 'goreleaser' (default) or 'goreleaser-pro'
				distribution: "goreleaser"
				version:      "latest"
				workdir:      "cmd/hof"
				args:         "release --clean -f goreleaser.yml -p 1"
			}
			env: {
				GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
			}
		}
	}

	buildx: {
		qemu: {
			name: "Set up QEMU"
			uses: "docker/setup-qemu-action@v2"
			with: {
				platforms: "arm64"
			}
		}

		setup: {
			linux: {
				name: "Set up Docker BuildX"
				uses: "docker/setup-buildx-action@v2"
			}
			macos: {
				name: "Set up Docker Colima"
				run: """
					brew install docker colima
					colima start --cpu 1 --memory 2 --disk 10
					"""
			}
		}

		login: {
			name: "Login to Docker Hub"
			uses: "docker/login-action@v2"
			with: {
				username: "${{ secrets.HOF_DOCKER_USER }}"
				password: "${{ secrets.HOF_DOCKER_TOKEN }}"
			}
		}

		formatters: {
			name: "Build Image"
			uses: "docker/build-push-action@v3"
			with: {
				context:   "formatters/tools/${{ matrix.formatter }}"
				file:      "\(context)/Dockerfile.debian"
				platforms: "linux/amd64,linux/arm64"
				tags:      strings.Join([
						"ghcr.io/hofstadter-io/fmt-${{ matrix.formatter }}:${{ env.HOF_SHA }}",
						"ghcr.io/hofstadter-io/fmt-${{ matrix.formatter }}:${{ env.HOF_TAG }}",
				], ",")
			}
			env: {
				GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
			}
		}
	}

	docker: {
		setup: {
			name: "Set up Docker"
			uses: "crazy-max/ghaction-setup-docker@v1"
			with: {
				version: "v23.0.1"
			}
			"if": "${{ startsWith( runner.os, 'macos') }}"
		}
		macos: {
			name: "Setup Docker MacOS var"
			run: """
				echo "DOCKER_HOST=\"unix://$HOME/.colima/default/docker.sock\"" >> $GITHUB_ENV
				"""
			"if": "${{ startsWith( runner.os, 'macos') }}"
		}

		login: {
			name: "Login to Docker Hub"
			uses: "docker/login-action@v2"
			with: {
				registry: "ghcr.io"
				username: "${{ github.actor }}"
				password: "${{ secrets.GITHUB_TOKEN }}"
			}
		}

		compat: {
			name: "Test Compatibility"
			run: """
				docker version
				docker info
				docker context ls
				go run test/docker/main.go
				"""
		}
	}

	gcloud: {
		auth: {
			name: "GCloud Auth"
			uses: "google-github-actions/auth@v1"
			with: credentials_json: "${{ secrets.HOF_GCLOUD_JSON }}"
		}
		setup: {
			name: "GCloud Setup"
			uses: "google-github-actions/setup-gcloud@v1"
		}

		dockerAuth: {
			name: "Docker Auth"
			run: """
				gcloud auth configure-docker
				"""
		}
	}

	hof: {
		install: {
			name: "Build hof"
			run:  "go install ./cmd/hof"
		}
	}

	docs: {

		setup: {
			name: "Setup"
			run: """
				hof fmt start prettier@v0.6.8-beta.12
				cd docs
				hof mod link
				make tools
				make deps
				"""
		}

		env: {
			name: "Docs Env"
			run: """
				D="next"
				[[ "$HOF_TAG" =~ ^docs-20[0-9]{6}.[0-9]+$ ]] && D="prod"
				echo "DOCS_ENV=${D}" >> $GITHUB_ENV
				"""
		}

	}
}
