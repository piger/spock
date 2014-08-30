# Spock wiki

Spock is a simple wiki software heavily inspired by [GitHub's Gollum](Gollum) and is mostly intended as a personal documentation system. I wrote it as a frontend for my technical notes.

[Gollum]: https://github.com/gollum/gollum

**Please note**: This is alpha software and should be used with caution!

## Why another wiki software?

There are many things I don't like about existing wiki softwares:

- CamelCase names are ugly
- most wiki softwares uses awful markup languages from ten years ago
- searching is usually painful
- editing wiki pages must be done from the web interface **only**, while I usually prefer to use my text editor
- you end up locked in within your wiki storage system (usually some gigantic RDBMS)
- you usually need a full LAMP stack to run a wiki

## Features

- wiki pages can be written in Markdown or RestructuredText and can be edited with your preferred text editor
- git is used as the underlying storage system
- full text search using a simple Python search server and [Whoosh](Whoosh)

[Whoosh]: https://pythonhosted.org/Whoosh/

## Installation

Requirements:

- recent version of Go1 (tested on Go 1.3)
- Python 2.6+ (2.7 is better)
  - [docutils](docutils) (for `rst` rendering)
- a C compiler

### Building Spock

To build Spock you first need to build a specific version of [libgit2](libgit2) along with `git2go`:

```
go get -d github.com/piger/git2go
cd $GOPATH/src/github.com/piger/git2go
git submodule update --init
make install
```

**NOTE**: `make install` will only build `git2go`, statically linking [libgit2](libgit2).

Now you can build Spock:

```
go get github.com/piger/spock
```

For the search server you better use a virtualenv:

```
virtualenv /usr/local/lib/spock/env
cd $GOPATH/src/github.com/piger/index
/usr/local/lib/spock/env/bin/python setup.py install
```

To render `RestructuredText` pages you will also need the `rst2html` program, included in [docutils](docutils) Python package; `rst2html` must be present in `$PATH`:

```
/usr/local/lib/spock/env/bin/pip install docutils
```

## Author

Daniel Kertesz <daniel@spatof.org>

[libgit2]: https://libgit2.github.com/

[git2go]: https://github.com/libgit2/git2go

[docutils]: http://docutils.sourceforge.net/
