# Spock wiki

Spock is a simple wiki software heavily inspired by
[GitHub's Gollum][Gollum] and is mostly intended as a personal
documentation system. I wrote it as a frontend for my technical notes.

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

- wiki pages can be written in Markdown or RestructuredText and can be
  edited with your preferred text editor
- git is used as the underlying storage system
- full text search (**beta**)
- nice browser editor thanks to [CodeMirror](http://codemirror.net)

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

## Installing Spock from sources

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

Note: you cannot use the `libgit2` package provided by your package manager because
`git2go` needs a specific version of the library.

On a Debian based GNU/Linux system you should be able to install all the
required dependencies running:

```bash
sudo apt-get install python-docutils cmake git mercurial libicu-dev libleveldb-dev
```

On OS X with [homebrew](http://brew.sh):

```bash
brew install mercurial cmake icu4c pkg-config leveldb
```

### Building Spock

To build Spock you first need to build `git2go` linking to a specific
version of [libgit2][libgit2]

```bash
go get -d github.com/libgit2/git2go
cd $GOPATH/src/github.com/libgit2/git2go
git submodule update --init
make install
```

**NOTE**: `make install` will only build `git2go`, statically linking [libgit2][libgit2]

Now you can build Spock by running `make`; if you system doesn't ship with an updated
version of `libleveldb` you can edit the `Makefile` and remove `leveldb` from the Go
build tags.

To render `RestructuredText` pages you will also need the `rst2html`
program, included in [docutils][docutils] Python package; `rst2html`
must be present in `$PATH`:

```bash
sudo pip install docutils
# or
sudo easy_install docutils
```

## Author

Daniel Kertesz <daniel@spatof.org>

Spock includes a copy of [CodeMirror 4.5](http://codemirror.net/): https://github.com/marijnh/codemirror 

[libgit2]: https://libgit2.github.com/

[git2go]: https://github.com/libgit2/git2go

[docutils]: http://docutils.sourceforge.net/
