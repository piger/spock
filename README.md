# Spock wiki

Spock is a simple wiki software heavily inspired by
[GitHub's Gollum][Gollum] and is mostly intended as a personal
documentation system. I wrote it as a frontend for my technical notes.

[Gollum]: https://github.com/gollum/gollum

**Please note**: This is alpha software and a toy project: use with caution!

## Why another wiki software?

- I don't want to fight with runtime dependencies, virtualenvs and such.
- I have a bunch of notes written in Markdown and RestructuredText formats.
- I like to edit wiki pages with my text editor.
- I like to use git to maintain the history of my notes.
- I don't want to run a full LAMP stack just to use my wiki.
- I'm having some fun learning Go.

## Features

- wiki pages can be written in Markdown or RestructuredText and can be
  edited with your preferred text editor
- git is used as the underlying storage system
- full text search (*experimental*)
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

- recent version of Go (tested with Go 1.3+)
- a C compiler
- git (to fetch some go dependencies)
- mercurial (to fetch some go dependencies)
- libgit 0.22.x
- icu4c
- leveldb (optional)
- Python [docutils][docutils] (optional, used for `rst` rendering)

On a Debian based GNU/Linux system you should be able to install
almost all the required dependencies running:

```bash
sudo apt-get install python-docutils cmake git mercurial libicu-dev libleveldb-dev
```

You must build libgit2 0.22.x by yourself.

On OS X with [homebrew](http://brew.sh):

```bash
brew install mercurial icu4c leveldb libgit2
```

### Building Spock

You can build Spock by running `make`; if you system doesn't ship with
an updated version of `libleveldb` you can edit the `Makefile` and
remove `leveldb` from the Go build tags.

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
