# drone-buildx-gcr

## Build

Build the binaries with the following commands:

```console
go build ./cmd/drone-buildx-gcr
./drone-buildx-gcr

//gar
go build ./cmd/drone-buildx-gar
./drone-buildx-gar
```

## Usage

### Running from the CLI

```console
export PLUGIN_TAG=latest
export PLUGIN_REPO=octocat/hello-world
export DRONE_COMMIT_SHA=d8dbe4d94f15fe89232e0402c6e8a0ddf21af3ab
./drone-buildx-gcr
```

## Release procedure

Create your pull request for the release. Get it merged then tag the release. The release will be published and changelog will be generated.

