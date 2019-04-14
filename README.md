## Usage

Run a server that will upload SHA files to a git repo for file backing

```
go run ./sget server https://github.com/philips/releases-test test
```

Download a release, generate the SHA file, and post it to the server

```
go run ./sget release-github -t v1.0 -o philips -r releases-test
```

