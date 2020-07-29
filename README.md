# rget

**Archived Project Warning**: rget is archived. Architectual issues made the project unmaintainable longterm ([see issue](https://github.com/merklecounty/rget/issues/42)). The rearchitected spiritual successor is under development: see [transparencylog/btget](https://github.com/transparencylog/btget).

`rget` downloads URLs and verifies the contents against a publicly recorded cryptographic log. The public log gives users of rget a number of useful properties:

- Verifiability of a downloaded URL's contents being identical to what the rest of the world sees
- Searchability of recorded content changes of a URL
- Notifications to any interested party about changes to the URLs contents

In practice the way the system works is a URL owner will publish the cryptographic digests at a URL adjacent to the content a `rget` user is downloading. The `rget` tool will download the digest and verify this digest appears in the Certificate Transparency log via a specially crafted DNS name.

Learn more and stay up to date with the [project blog and newsletter](https://merklecounty.substack.com/). Checkout some of the blog posts:

- [rget: a secure download user story](https://merklecounty.substack.com/p/rget-a-secure-download-user-story)

## Installation

Download the appropriate release from https://github.com/merklecounty/rget/releases and extract the archive

## Example Usage

Use rget to download the v3.4.2 [etcd](https://etcd.io) release for macOS and verify that the contents are publicly recorded.

```
rget https://github.com/etcd-io/etcd/releases/download/v3.4.2/etcd-v3.4.2-darwin-amd64.zip
```

## Developer Usage

### GitHub Developer Usage

It takes two commands to make a release available for verified download with rget:

```
rget github publish-release-sums https://github.com/merklecounty/rget/releases/tag/v0.0.6
```

The first command will start a local web server and open a Github authorization URL
in your browser. You will have 120 seconds to authorize it.

When authorized, it will calculate SHA256 sums for every file in the release,
create a `SHA256SUMS` file, and add it to the Github release.

```
rget submit https://github.com/merklecounty/rget/releases/download/v0.0.6/SHA256SUMS
```

The second command will submit the sums to the log. This does not use any GitHub credentials.

**Note:** If a project has release automation that uploads to GitHub simply add
the creation of SHA256SUMS to the automation instead of using `github
publish-release-sums` and call `rget submit` after uploading. See the
[integrations doc](Documentation/integrations.md) for a list of tools that work
out of the box. As an example `rget` uses [Go
Releaser](https://goreleaser.com/) for automation.

## Administration Usage

Run a server that will upload SHA files to a git repo for file backing

```
rget server <public git repo> <private certificates git repo>
```

## FAQ

If you have a question that isn't answered here please [open an issue](https://github.com/merklecounty/rget/issues/new) or [start a discussion on the mailing list](https://groups.google.com/forum/#!forum/rget)

- **Q**: Where did this idea come from?
- **A**: This project builds upon a design doc for [Binary Transparency](https://wiki.mozilla.org/Security/Binary_Transparency) originally designed for the Mozilla Firefox project. 

- **Q**: Why not build this ontop of blockchain?
- **A**: Blockchain could be used to create a similar guarantee. However, using the Certificate Transparency technology extends a number of advantages and was a pragmatic choice to get this project going: the industry momentum of certificate transparency log technology [(1)](https://ct.cloudflare.com/about), leverage existing web technologies like DNS and TLS [(2)](https://www.certificate-transparency.org/how-ct-works), and finally most practical applications that want to use blockchain with the web end up using a centralized gateway for speed and reliability [(3)](https://blog.cloudflare.com/cloudflare-ethereum-gateway/)[(4)](https://infura.io/docs/ethereum/json-rpc/eth_blockNumber). Perhaps as the bridge between the web and blockchain matures it will become a more practical option.

- **Q**: Why not use GPG keys or other public key signing?
- **A**: This is complimentary to public key signing! Public key signing asserts that someone with access to the private key signed the exact content. But, the private key can be used to generate an unlimited number of signatures for different content. If the URLs contents are both signed and logged in the URL content record then there is a guarantee that both the owner of the private key signed the content AND the content being fetched is cryptographically identical to the content other people are fetching using rget.

- **Q**: Where does the name rget come from?
- **A**: The "r" stands for recorder, as in a clerk who records or processes records. In many governments a Recorder of Deeds (aka Registrar General, County Clerk, etc) is an official who is tasked with recording and maintaining important public records of real property. Similarly this project aims to maintain a public record of internet property in the form of the cryptographic digest of certain URLs and provide tools to verify those records. The "get" comes from the HTTP GET verb and other tools like Wget.

- **Q**: What are examples of practical attacks this could mitigate?
- **A**: A well known example is the Feb. 2016 attack on the Linux Mint project where an attacker replaced a version of a Linux Mint release with a new version that included a backdoor vulnerability. With luck this was detected and mitigated within a day, however, there are likely many projects that have been attacked in a similar way without catching the attack. Further, the project could not make a strong assurance to the community on how long they were vulnerable, only stating "As far as we know, the only compromised edition was Linux Mint 17.3 Cinnamon edition.". By ensuring the cryptographic digests of all releases end up in a publicly audited log the project could have stated exactly when the content changed and potentially used a Certificate Transparency monitor to get notified quickly once it happened.

- **Q**: What happens if an attacker can modify SHA256SUMS files?
- **A**: The modification will show up in the logs. As an example the v1.0 release of the philips/releases-test project was modified several times. And this appears in the log on both [crt.sh](https://crt.sh/?q=%.v1-0.releases-test.philips.github.com.recorder.merklecounty.com) and [Google's Transparency Report](https://transparencyreport.google.com/https/certificates?hl=en&cert_search_auth=&cert_search_cert=&cert_search=include_expired:false;include_subdomains:true;domain:v1-0.releases-test.philips.github.com.recorder.merklecounty.com&lu=cert_search)
