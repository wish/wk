SHA  := $(shell git rev-parse --short HEAD)
DATE := $(shell date +"%a %b %d %T %Y")

build:
	@echo "$@"
	@CGO_ENABLED=0 go build -ldflags \
	       '-X "github.com/wish/wk/cmd/wk/cmd.BuildDate=${DATE}" -X "github.com/wish/wk/cmd/wk/cmd.BuildSha=${SHA}"' \
			github.com/wish/wk/cmd/wk
