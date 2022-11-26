yage
====

`yage` is a fork of `filippo.io/age/cmd/age` with added YAML support.

This project contains **no cryptographic logic**, all of that remains
[in the original project](https://github.com/FiloSottile/age).

`yage` encrypts YAML key values in place using YAML tag `!crypto/age` as marker.
It only support encoding strings.

Tag / attributes
----------------

```yaml
---
simpletag: !crypto/age simple value
doublequoted: !crypto/age:DoubleQuoted double quoted value
singlequoted: !crypto/age:SingleQuoted single quoted value
literal: !crypto/age:Literal literal value
flowed: !crypto/age:Flow flowed value
folded: !crypto/age:Folded folded value
notag: !crypto/age:Literal,NoTag literal untagged value # the NoTag attribute will cause yage to drop the tag when decrypting
```

Example
-------

```yaml
simpletag: !crypto/age simple value
```

```shell
$ yage --encrypt --yaml -R ~/.ssh/id_ed25519.pub < simple.yaml
simpletag: !crypto/age |-
  -----BEGIN AGE ENCRYPTED FILE-----
  YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IHNzaC1lZDI1NTE5IEcwQmFrQSBHdk9o
  V3dDbTRSNlVuei82RDJlRnNaMnduTWpLSkZEbVlJdmdUdDdJNjJvCkVZdDZ6cTRu
  QWplUythdERuNldlTzJMR0p2VjI3UGx2OWt4Q3VaMDZXK0kKLS0tIG9ZZTZ4K2FM
  c2VKVXlLamJndE1JaDN5SkdwTjEyR0FIeXFHTEZDZGZWSGcKclDEC1Xo41AdhLa2
  rbzwJeC4KyynjhJbOvwRlCBJV6K479LbfLSicgKjk9g=
  -----END AGE ENCRYPTED FILE-----
```

⚠️ YAML formatting may be modified when encrypting/decrypting in place due to
limitations of the YAML library used. If you must conserve YAML formatting
you'll need to encrypt it as a regular file.

```
$ yage encrypt --yaml -R ~/.ssh/id_ed25519.pub -R ~/.ssh/someone@devnull.io.pub file.yaml > file.yaml.age
$ yage decrypt --yaml -i ~/.ssh/id_ed25519 file.yaml.age > file.yaml
$ yage rekey --yaml -i ~/.ssh/id_ed25519 -R ~/.ssh/id_ed25519.pub -R ~/.ssh/someone+else@devnull.io.pub file.yaml.age
```

Install
-------

### From sources

```shell
$ mkdir ~/git && cd ~/git
$ git clone git@github.com:sylr/yage.git
$ cd yage && make install
```

### Binaries

You can find pre-built binaries in the [here](https://github.com/sylr/yage/releases).
