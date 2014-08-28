from setuptools import setup, find_packages


setup(
    name="vance",
    version="0.1",
    description="Small search engine",
    author="Daniel Kertesz",
    author_email="daniel@spatof.org",
    install_requires=[
        'Flask',
        'Whoosh>=2.6.0',
        'PyYAML',
    ],
    setup_requires=[],
    tests_require=[],
    zip_safe=False,
    include_package_data=True,
    packages=find_packages(),
    entry_points={
        'console_scripts': [
            'vance = vance.main:main',
        ],
    },
)
