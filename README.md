# party-dl
Download images and videos from the .party sites (coomer.party & kemono.party).
This tool can also add the metadata to [Stash](https://stashapp.cc/).

# Usage
Download content
```sh
$ party-dl download {URL}
$ party-dl download --base-location ./output {URL}
```

Add metadata to stash
```sh
$ party-dl stash --stash-host http://localhost:9999 --content ./data/
```