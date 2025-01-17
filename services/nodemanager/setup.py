#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import
import os
import sys
import re

from setuptools import setup, find_packages

SETUP_DIR = os.path.dirname(__file__) or '.'
README = os.path.join(SETUP_DIR, 'README.rst')

import arvados_version
version = arvados_version.get_version(SETUP_DIR, "arvnodeman")
if os.environ.get('ARVADOS_BUILDING_VERSION', False):
    pysdk_dep = "=={}".format(version)
else:
    # On dev releases, arvados-python-client may have a different timestamp
    pysdk_dep = "<={}".format(version)

short_tests_only = False
if '--short-tests-only' in sys.argv:
    short_tests_only = True
    sys.argv.remove('--short-tests-only')

setup(name='arvados-node-manager',
      version=version,
      description='Arvados compute node manager',
      long_description=open(README).read(),
      author='Arvados',
      author_email='info@arvados.org',
      url="https://arvados.org",
      license='GNU Affero General Public License, version 3.0',
      packages=find_packages(),
      scripts=['bin/arvados-node-manager'],
      data_files=[
          ('share/doc/arvados-node-manager', ['agpl-3.0.txt', 'README.rst', 'arvados-node-manager.service']),
      ],
      install_requires=[
          'apache-libcloud>=2.3.1.dev1',
          'arvados-python-client{}'.format(pysdk_dep),
          'future',
          'pykka < 2',
          'python-daemon',
          'setuptools',
          'subprocess32>=3.5.1',
      ],
      dependency_links=[
          "https://github.com/curoverse/libcloud/archive/apache-libcloud-2.3.1.dev1.zip"
      ],
      test_suite='tests',
      tests_require=[
          'requests',
          'pbr<1.7.0',
          'mock>=1.0',
          'apache-libcloud>=2.3.1.dev1',
          'subprocess32>=3.5.1',
      ],
      zip_safe=False
      )
