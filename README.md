## Usage

Run a server that will upload SHA files to a git repo for file backing

```
go run ./sget server https://github.com/philips/releases-test test
```

Download a release, generate the SHA file, and post it to the server

```
go run ./sget release-github -t v1.0 -o philips -r releases-test
```


## First time it worked!

This README entry should be deleted one day. But, just for fun here is the first time it worked.

```
sget https://api.github.com/repos/philips/releases-test/zipball/v2.0
Retrieve certificate chain from TLS connection to "5c793eed08df469f8a40a0465f767677.a73ecbf830829713e0037efd9b1a357e.secured.ifup.org:443"
Found chain of length 2
Found 0 external SCTs for "https://5c793eed08df469f8a40a0465f767677.a73ecbf830829713e0037efd9b1a357e.secured.ifup.org", of which 0 were validated
Examine embedded SCT[0] with timestamp: 1558561352793 (2019-05-22 14:42:32.793 -0700 PDT) from logID: e2694bae26e8e94009e8861bb63b83d43ee7fe7488fba48f2893019dddf1dbfe
Validate embedded SCT[0] against log "DigiCert Yeti2019 Log"...Validate embedded SCT[0] against log "DigiCert Yeti2019 Log"... validated
Check embedded SCT[0] inclusion against log "DigiCert Yeti2019 Log"...
Check embedded SCT[0] inclusion against log "DigiCert Yeti2019 Log"... included at 172455296
Examine embedded SCT[1] with timestamp: 1558561352327 (2019-05-22 14:42:32.327 -0700 PDT) from logID: 63f2dbcde83bcc2ccf0b728427576b33a48d61778fbd75a638b1c768544bd88d
Validate embedded SCT[1] against log "Google 'Argon2019' log"...Validate embedded SCT[1] against log "Google 'Argon2019' log"... validated
Check embedded SCT[1] inclusion against log "Google 'Argon2019' log"...
Check embedded SCT[1] inclusion against log "Google 'Argon2019' log"... included at 490233137
Found 2 embedded SCTs for "5c793eed08df469f8a40a0465f767677.a73ecbf830829713e0037efd9b1a357e.secured.ifup.org", of which 2 were validated
Expecting sha256sum 4e1e860029252e3d4b953161500eae538b295d52bc1760e90e17435415441c78 for https://api.github.com/repos/philips/releases-test/zipball/v2.0
Download validated and saved to philips-releases-test-v2.0-0-gad0a78a.zip
```
