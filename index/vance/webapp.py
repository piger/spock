import os
import codecs
from flask import Flask, request, jsonify, abort
from vance.search import Index, Document


app = Flask(__name__)
index = Index()


@app.route('/api/search', methods=['POST'])
def search():
    data = request.json
    if not data:
        app.logger.error("empty request.json")
        abort(400)

    query = data.get('query')
    if not query:
        app.logger.error("empty query")
        abort(400)

    results, suggestion = index.search(query)
    return jsonify(result={ 'status': 'ok', 'results': results,
                            'suggestion': suggestion })

@app.route('/api/find', methods=['POST'])
def find():
    data = request.json
    if not data:
        app.logger.error("empty request.json")
        abort(400)

    query = data.get('name')
    if not query:
        app.logger.error("empty query")

    results = index.find_document(query)
    return jsonify(result={ 'status': 'ok', 'results': results })


@app.route('/api/get', methods=['POST'])
def get_document():
    data = request.json
    if not data:
        app.logger.error("empty request.json")
        abort(400)

    filename = data.get('path')
    document = index.get_document(filename)
    if document is None:
        abort(404)

    rv = {
        'id': document['title'],
        'lang': document['lang'],
    }
    return jsonify(result={ 'status': 'ok', 'document': rv })


@app.route('/api/add', methods=['POST'])
def add():
    data = request.json
    if not data:
        app.logger.error("add: empty request.json")
        abort(400)

    filename = data.get('name')
    lang = data.get('lang')
    if not filename:
        app.logger.error("add: missing 'name'")
        abort(400)

    if filename.startswith('/'):
        filename = filename[1:]

    doc = index.read_document(filename)
    if lang:
        doc.set_lang(lang)
    index.add([doc])

    return jsonify(result={'status': 'ok'})
