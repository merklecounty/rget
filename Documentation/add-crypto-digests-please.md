# rget project request template

To use `rget` projects need to publish SHA256SUMS files. This is a template on how to politely request that those files be created.

Subject: Add Cryptographic Digests to GitHub Releases
Body:

Please consider adding cryptographic digests for the [files released in this project][releases]. Commonly called SHA256SUMS files they can be easily generated using the common `sha256sum` tool on most systems

```
sha256sum * > SHA256SUMS
```

Alternatively, there [are some release automation tools][integrations] that can build these files automatically.

Besides being a useful practice for download verification I would also like to use the SHA256SUMS as a way to ensure the releases aren't tampered with and track when they are modified. There is a [tool called rget][rget] that can do this if you provide SHA256SUMS for your releases.

The rget tool also has a subcommand to make it easy to create SHA256SUMS for existing releases, just run:

```
rget github publish-release-sums https://github.com/REPLACE_ORG/REPLACE_PROJ/releases/tag/v2.0
```

Thanks!

[releases]: https://github.com/REPLACE_ORG/REPLACE_PROJ/releases
[integrations]: https://github.com/merklecounty/rget/blob/master/Documentation/integrations.md#release-automation
[rget]: https://github.com/merklecounty/rget#rget
