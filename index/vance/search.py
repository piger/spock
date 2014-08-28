import os
import yaml
from whoosh.index import create_in, open_dir, exists_in
from whoosh.fields import *
from whoosh.qparser import QueryParser, DisMaxParser
from whoosh.analysis import LanguageAnalyzer
from whoosh.writing import AsyncWriter
import codecs


schema = Schema(title=ID(stored=True, unique=True),
                name=NGRAM(field_boost=1.5),
                lang=STORED,
                content_it=TEXT(lang="it", spelling=True),
                content_en=TEXT(lang="en", spelling=True))

SEARCH_FIELDS = {
    'name': 1.5,
    'content_it': 1.0,
    'content_en': 1.0,
}

PAGE_EXTS = ( "md", "txt", "rst" )


def _extract_name(path):
    basename = os.path.basename(path)
    return basename.rsplit('.', 1)[0]

class Document(object):
    def __init__(self, title, content, header):
        self.title = title
        self.content = content
        self.header = header
        self.name = _extract_name(self.title)

    @classmethod
    def read_from_file(cls, title, filename):
        with codecs.open(filename, 'rb', 'utf-8') as fd:
            data = fd.read()

        if data.startswith(u"---"):
            mark = data.find(u"---", 3)
            if mark == -1:
                content = data[:]
                header = {}
            else:
                headers = yaml.load_all(data[0:mark+7])
                header = list(headers)[0]
                content = data[mark+8:]
        else:
            header = {}
            content = data[:]

        doc = Document(title, content, header)
        return doc 

    def get_lang(self):
        return self.header.get('language', '[UNKNOWN]')

    def has_lang(self):
        return 'language' in self.header

    def set_lang(self, lang):
        self.header['language'] = lang

    def serialize(self):
        rv = {
            'title': self.title,
            'name': self.name,
            'lang': self.get_lang(),
        }

        if self.has_lang():
            content_key = 'content_' + self.get_lang()
            rv[content_key] = self.content
        else:
            rv['content_it'] = self.content
            rv['content_en'] = self.content
        return rv


class Index(object):
    def __init__(self):
        self.db_path = None
        self.repo_path = None

    def open_index(self, db_path):
        self.db_path = db_path

        if not os.path.exists(self.db_path):
            os.makedirs(self.db_path, mode=0775)

        if exists_in(self.db_path):
            self.ix = open_dir(self.db_path, schema=schema)
        else:
            self.ix = create_in(self.db_path, schema)

    def read_document(self, filename):
        real_filename = os.path.join(self.repo_path, filename)
        return Document.read_from_file(filename, real_filename)

    def add(self, documents, async=True):
        """Add many Document(s) to the index"""

        rv = 0

        if async:
            writer = AsyncWriter(self.ix)
        else:
            writer = self.ix.writer()
        
        for document in documents:
            writer.update_document(**document.serialize())
            rv += 1

        writer.commit()
        return rv

    def search(self, qstring):
        """Perform a search and returns a tuple containing the search
        results and an optional suggestion"""

        rv = []
        suggestion = None

        qp = DisMaxParser(SEARCH_FIELDS, self.ix.schema)
        query = qp.parse(qstring)

        with self.ix.searcher() as searcher:
            results = searcher.search(query, terms=True)
            corrected = searcher.correct_query(query, qstring)
            if corrected.query != query:
                suggestion = corrected.string

            for result in results:
                title = result['title']
                lang = result.fields().get('lang')

                filename = os.path.join(self.repo_path, title)
                doc = Document.read_from_file(title, filename)

                hl_it = result.highlights("content_it", text=doc.content)
                hl_en = result.highlights("content_en", text=doc.content)

                if hl_it != hl_en:
                    hl = "\n".join([hl_it, hl_en])
                else:
                    hl = hl_it

                r = {
                    'title': title,
                    'lang': lang,
                    'highlight': hl,
                }
                rv.append(r)

        return (rv, suggestion)

    def get_document(self, title):
        """Return a document identified by title"""

        with self.ix.searcher() as searcher:
            rv = searcher.document(title=title)
            return rv

    def find_document(self, title):
        qp = QueryParser("name", self.ix.schema)
        query = qp.parse(title)
        rv = []

        with self.ix.searcher() as searcher:
            results = searcher.search(query)
            for result in results:
                rv.append(result['title'])

        return rv
                
