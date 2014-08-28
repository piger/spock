import shlex
import subprocess
import codecs
import os
from optparse import OptionParser

from vance.webapp import app, index
from vance.search import Index, Document, PAGE_EXTS


def create_app(cfg={}):
    app.config.update(cfg)
    app.config.from_envvar('APP_CONFIG', silent=True)

    index.open_index(app.config['INDEX_DIR'])
    index.repo_path = app.config['REPO_DIR']

    return app


def walk_documents(filenames, index):
    for filename in filenames:
        filename = unicode(filename.strip())
        if not filename:
            continue

        name, ext = os.path.splitext(filename)
        if not ext:
            continue

        if ext.startswith("."):
            ext = ext[1:]
        if ext in PAGE_EXTS:
            doc = index.read_document(filename)
            yield doc

def main():
    parser = OptionParser()
    parser.add_option('-d', '--db', help="Path to index directory")
    parser.add_option('-r', '--repo', help="Path to the repository")
    parser.add_option('-D', '--debug', action='store_true')
    parser.add_option('--web', action='store_true')
    parser.add_option('--index', action='store_true')

    opts, args = parser.parse_args()

    if not opts.db or not opts.repo:
        parser.error("You must specify both --db and --repo")

    if opts.web:
        cfg = {
            'INDEX_DIR': opts.db,
            'REPO_DIR': opts.repo,
        }
        _app = create_app(cfg)
        _app.run(debug=opts.debug)
    elif opts.index:
        index = Index()
        index.open_index(opts.db)
        index.repo_path = opts.repo

        cmd = shlex.split("git --git-dir %s/.git ls-tree -r master --name-only" % (
            opts.repo))
        output = subprocess.check_output(cmd)
        filenames = output.split("\n")
        rv = index.add(walk_documents(filenames, index), async=False)
        print "Indexed %d documents" % rv


if __name__ == '__main__':
    main()
