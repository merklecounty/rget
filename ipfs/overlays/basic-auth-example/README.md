# Basic Auth Example

To add users and their passwords, run the following and then add it to the `secret.yaml` file:

```sh
$ echo "admin:muchlongpassword,alice:verypassword,bob:manysecure" | base64 -w 0 -
YWRtaW46bXVjaGxvbmdwYXNzd29yZCxhbGljZTp2ZXJ5cGFzc3dvcmQsYm9iOm1hbnlzZWN1cmUK⏎
```

The `secret.yaml` should look like the following when you are done:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: secret-config
type: Opaque
data:
  basicauthcreds: YWRtaW46bXVjaGxvbmdwYXNzd29yZCxhbGljZTp2ZXJ5cGFzc3dvcmQsYm9iOm1hbnlzZWN1cmUK
```
