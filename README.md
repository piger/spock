# Spock wiki

Spock is a simple wiki software heavily inspired by [GitHub's Gollum][Gollum] and is mostly intended as a personal documentation system. I wrote it as a frontend for my technical notes.

[Gollum]: https://github.com/gollum/gollum

**Please note**: This is alpha software and should be used with caution!

## Why another wiki software?

- I don't want to fight with runtime dependencies, virtualenvs and such.
- I have a bunch of notes written in Markdown and RestructuredText formats.
- I like to edit wiki pages with my text editor.
- I like to use git to maintain the history of my notes.
- I don't want to run a full LAMP stack just to use my wiki.
- I'm having some fun with Go.

## Features

- wiki pages can be written in Markdown or RestructuredText and can be edited with your preferred text editor
- git is used as the underlying storage system
- full text search (**beta**)

**NOTE**: RestructuredText is **not** rendered by Go code, see below.

## Usage

The first time you launch Spock it will need to create the repository directory:

```bash
./spock -repo ~/Documents/wiki -init
```

Typical usage:

```bash
./spock -repo ~/Documents/wiki
```

## Building Spock

Requirements:

- recent version of Go1 (tested on Go 1.3)
- a C compiler
- cmake (to build libgit2)
- pkg-config (to build libgit2)
- git (to fetch some go dependencies)
- mercurial (to fetch some go dependencies)
- Go 1.x (tested on Go 1.3)
- icu4c
- leveldb (optional)
- Python [docutils][docutils] (optional, used for `rst` rendering)

On a Debian based GNU/Linux system you should be able to install all the
required dependencies running:

```bash
sudo apt-get install python-docutils cmake git mercurial libicu-dev
```

On OS X with [homebrew](http://brew.sh):

```bash
brew install mercurial cmake icu4c pkg-config libleveldb-dev
```

You will also need to download a copy of [CodeMirror][CodeMirror] and unpack it
inside the `data/static/` directory, so that you end up having:

```
$ ls data/static/
codemirror-4.5/  css/  favicon.ico  fonts/  js/
```

[CodeMirror]: http://codemirror.net/codemirror.zip

### Building Spock

To build Spock you first need to build a specific version of [libgit2][libgit2] along with `git2go`:

```bash
go get -d github.com/piger/git2go
cd $GOPATH/src/github.com/piger/git2go
git checkout dev
git submodule update --init
make install
```

**NOTE**: `make install` will only build `git2go`, statically linking [libgit2][libgit2].

Now you can build Spock (you can safely omit `leveldb` if your system doesn't ship with an up to date version of the library):

```bash
go get -d -tags "libstemmer icu leveldb" github.com/piger/spock
cd $GOPATH/src/github.com/piger/spock
# On GNU/Linux:
go build -tags "libstemmer icu leveldb" cmd/spock/spock.go
# On OS X:
./build-osx.sh
```

To render `RestructuredText` pages you will also need the `rst2html` program, included in [docutils][docutils] Python package; `rst2html` must be present in `$PATH`:

```bash
sudo pip install docutils
# or
sudo easy_install docutils
```

### Packing a Linux distribution-independent archive

To pack an archive containing a distribution-independent GNU/Linux version of Spock:

```bash
mkdir -p spock-linux-i386/lib/i386-linux-gnu
for lib in `ldd spock | awk '$3 ~ /^\// { print $3 }'`; do cp $lib spock-linux-i386/lib/i386-linux-gnu/; done
cp /lib/ld-linux.so.2 spock-linux-i386/lib/ld-linux-i386
tar --xz -cvf spock-linux-i386.tar.xz spock-linux-i386
```

## Author

Daniel Kertesz <daniel@spatof.org>

[libgit2]: https://libgit2.github.com/

[git2go]: https://github.com/libgit2/git2go

[docutils]: http://docutils.sourceforge.net/
