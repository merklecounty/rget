# Cluster Secret
To generate the `cluster_secret` value in `secret.yaml`, run the following and insert the output in the appropriate place in the `secret.yaml` file:

```sh
$ od  -vN 32 -An -tx1 /dev/urandom | tr -d ' \n' | base64 -w 0 -
```

# Bootstrap Peer ID and Private Key
To generate the values for `bootstrap_peer_id` and `bootstrap_peer_priv_key`, install [`ipfs-key`](https://github.com/whyrusleeping/ipfs-key) and then run the following:

```sh
$ ipfs-key | base64 -w 0
```

Copy the id into the `env-configmap.yaml` file. Then copy the private key value and run the following with it:

```sh
$ echo "<INSERT_PRIV_KEY_VALUE_HERE>" | base64 -w 0 -
```

Copy the output to the `secret.yaml` file.

