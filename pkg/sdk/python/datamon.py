from datamon import error, listRepos, listBundles
import json
from dumper import dump

#class Config(object):

def Repos():
  # TODO(fred): config here to be formalized with object...
  config = '{}'
  return json.loads(listRepos(config))

def Bundles(repo):
  config = '{}'
  return json.loads(listBundles(config,repo))

# Try
if __name__ == '__main__':
  repos = Repos()
  dump(repos);
  bundles = Bundles("migration-test-fred-2")
  dump(bundles);
  # TODO(fred): test error handling
  #      try:
  #      except error as e:
