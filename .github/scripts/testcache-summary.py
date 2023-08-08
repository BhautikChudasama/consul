#!/usr/bin/env python3

import collections
import sys
import re
import logging

# all known testcache lines
RE_TEST_ID = re.compile(r'testcache: (?P<package>[^:]+): test ID ([0-9a-f]+) => ([0-9a-f]+)')
RE_TEST_ID_INPUT_ID = re.compile(r'testcache: (?P<package>[^:]+): test ID ([0-9a-f]+) => input ID ([0-9a-f]+) => ([0-9a-f]+)')
RE_INPUT_LIST_NOT_FOUND = re.compile(r'testcache: (?P<package>[^:]+): input list not found: (.*)')
RE_INPUT_FILE_TOO_NEW = re.compile(r'testcache: (?P<package>[^:]+): input file (?P<file>.*): file used as input is too new')
RE_SAVE = re.compile(r'testcache: (?P<package>[^:]+): save test ID ([0-9a-f]+) => input ID ([0-9a-f]+) => ([0-9a-f]+)')

focus_package = None
# lazy argparse
if len(sys.argv) > 1:
  if sys.argv[1] in ["-h", "-help" , "--help"]:
    print("""usage: testcache-summary.py [package]""")
  focus_package = sys.argv[1]

lookups = collections.Counter()
saves = collections.Counter()
misses = collections.Counter() 

problems = collections.defaultdict(list)

for l in sys.stdin.readlines():
  if not l.startswith('testcache:'):
    continue
  l = l.strip()

  m = RE_TEST_ID.match(l)
  if m is not None:
    lookups[m.group("package")] += 1
    continue

  m = RE_TEST_ID_INPUT_ID.match(l)
  if m is not None:
    # don't need it
    continue

  m = RE_INPUT_LIST_NOT_FOUND.match(l)
  if m is not None:
    misses[m.group("package")] += 1
    continue

  m = RE_INPUT_FILE_TOO_NEW.match(l)
  if m is not None:
    problems[m.group("package")].append("'file too new %s' indicates uncacheable test" % (m.group("file"),))
    continue

  m = RE_SAVE.match(l)
  if m is not None:
    saves[m.group("package")] += 1
    continue

  logging.warning("Unhandled line: %r", l)

tl = 0
ts = 0
tm = 0

for p, l in lookups.items():
  if focus_package is not None and p != focus_package:
    continue
  try: 
    s = saves[p]
  except KeyError:
    s = 0

  try: 
    m = misses[p]
  except KeyError:
    m = 0
  h = l-m
  print("%s: hit rate %d/%d %.2f%%; saves %d" % (p, h, l, h*100/l, s))
  if m > 0 and s != m:
    problems[p].append("not all misses resulted in a save: %d/%d" % (s, m))

  tl += l
  ts += s
  tm += m

if focus_package is None:
  print("TOTAL: hit rate %d/%d %.2f%%; saves %d" % (tl-tm, tl, (tl-tm)*100/tl, ts))

if len(problems) > 0:
  print("\nPROBLEMS:")
  for p, vs in problems.items():
    print(p + ":")
    for v in vs:
      print("\t %s" % (v,))