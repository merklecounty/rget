## Usage

Download a GitHub releases URL and verify it against the certificate transparency log.

```
sget https://api.github.com/repos/philips/releases-test/zipball/v2.0
```

## Developer Usage

### GitHub Developer Usage

Generate SHA256SUMS and upload to the GitHub releases page

```
sget github publish-release-sums https://github.com/philips/releases-test/releases/tag/v2.0
```

### Submit a SHA256SUMS to the log

```
sget submit https://github.com/philips/releases-test/releases/download/v2.0/SHA256SUMS
```

## Administration Usage

Run a server that will upload SHA files to a git repo for file backing

```
sget server <public git repo> <private certificates git repo>
```
