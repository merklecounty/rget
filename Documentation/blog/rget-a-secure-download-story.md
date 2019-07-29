Attacks that introduce malicious modifications to software are seemingly on the rise. In the Open Source Software ecosystem a number of recent attacks have shown that seemingly trusted distribution points are at risk of distributing undetected modification to users. The general pattern is that an attacker gets access to code repositories, package managers, and/or file servers via a misplaced credential, weak password, or less commonly, an exploit. These attacks could happen to nearly any project but a few recent examples include [strong_password ruby gem](https://withatwist.dev/strong-password-rubygem-hijacked.html), [Canonical’s GitHub](https://twitter.com/dclauzel/status/1147525512794988544), [Purescript Installer](https://harry.garrood.me/blog/malicious-code-in-purescript-npm-installer/), [Linux Mint](https://blog.linuxmint.com/?p=2994). 

The goal of rget is to both help users and authors notice these attacks and confidently recover from them.

Before tackling all of those problems the story of rget will start simple. There is a user wanting to download some executable files. But, how do they trust the applications, scripts, or source code that will eventually execute on their computer? It is always risky. And all users needs to evaluate three key questions:

1. Is the author trustworthy? 
2. Is this the legitimate website of the author? 
3. Is the website providing the same download that the author intended everyone to receive? 

A sophisticated web user can potentially navigate the first two questions by relying on a number of technical and social hints such as:

- Domains and URL paths being https and at a familiar website 
- Following links provided by trusted friends on social media, email, or trusted news aggregators 
- Looking at social clues such as “likes/stars/download” counters on trusted websites 

In all of these cases people are looking for validation that the thing they are about to download is the same thing that someone else, who’s opinion they trust, either downloaded or published. But, how can you trust that a website is providing a trustworthy version of the download? How do you know you are getting the same copy of the software that your friend did? Or how does the author know their software is being delivered to users unaltered?

There are many potential solutions to this problem but the internet has relied on two primarily:

1. Public key cryptography to verify a file was signed by the author 
2. Cryptographic digests (e.g. SHA256SUMS) to verify a files contents were correctly received 

However, often the SHA256SUMS files and the author’s signature and public key are hosted in the same website as the file you want to trust. This presents a problem as a compromise of either the author’s credentials or the website itself could potentially lead to people receiving altered files.

**An Analogy**

Outside of the world of computers there are similar problems of asset trust and accuracy. For example, imagine someone is buying a land property from another person; there are some very important questions to figure out.

- How does the buyer know the seller actually owns the property? 
- How does the buyer ensure that the land is the size/shape/location described by the seller? 

In many countries the ownership, location, size, shape, and other details of land property are kept in public record logs. These record logs are often managed by a County Recorder’s office and the records are available to anyone inspect.

Wouldn’t it be useful if such a public record existed for applications and scripts on the internet?

**A Solution, Introducing rget**

rget is a command line application for Linux, Windows, and macOS that downloads a file from a URL and verifies the cryptographic digest of the file against a set of public logs operated by multiple organizations.

It is a sophisticated process. However, the user experience is familiar and simple:

rget https://github.com/philips/releases-test/archive/v1.0.zip

Running this command does a number of things in the background but the big feature of the logs used by rget are the same public logs that have been starting to secure the SSL ecosystem. The [Certificate Transparency](https://www.certificate-transparency.org/) (CT) logs provide strong uptime, audit, and third party indexing out of the box. If you are interested in learning how rget works behind the scenes it takes heavy inspiration from [this design doc](https://wiki.mozilla.org/Security/Binary_Transparency) by the Mozilla Firefox team.

If a tool like rget gets significant usage in industry I think it can put pressure on software publishers to participate. Today, it is a prototype of an idea but imagine a world where browsers give huge warnings when downloading executable files if they don’t publish the cryptographic digests to a log. It would be a good foundation for better web security just as certificate transparency was for the CA infrastructure.

[Try out rget today via the most recent release on GitHub](https://github.com/merklecounty/rget/releases).

**Next Steps**

Keep in touch with the project! Subscribe to the [email list](https://merklecounty.substack.com/) and get notified of new releases and blog posts.

If you are a software author please consider uploading SHA256SUMS for your files on GitHub. The rget command has an easy to use subcommand for adding these cryptographic digests to your existing releases. And if you already publish SHA256SUMS try out rget submit and get started today.

If you are a user please consider asking projects you rely on to publish SHA256SUMS on their GitHub releases. There is a template you can copy and paste when emailing software authors or filing a GitHub issue.

This project is just getting started and there is a lot of work to do. We would love help creating well-known protocols so any website can use rget, add support to rget for other providers like Bitbucket/GitLab/etc, and add support for packaging formats like OCI containers.
