# ssh-site

Snehanshn's portfolio, served over SSH.
The address will be `ssh snehanshn.duckdns.org` once it's live - it isn't yet.

## Running locally

```sh
make content   # fetch the content pack into internal/content/pack/
make run       # build and start the server on localhost:2222
```

Then, from another terminal:

```sh
ssh -p 2222 localhost
```

Press `q` to quit.

## Content

All facts about Snehanshn come from [github.com/SnehanshnC/content-pack](https://github.com/SnehanshnC/content-pack) at build time - nothing is hardcoded in this repo.

## Planning

Planning happens on a local wayfinder tracker kept out of this repo.
