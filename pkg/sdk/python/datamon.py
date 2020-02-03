from datamon import error, listRepos
import json
import functools

class Config(object):

class Repo(object):
  """
  Repo is a datamon repository
  """

  def __init__(self, app, config=None):
    self.config = config
    self.wsgi_app = create_wsgi_middleware(app.wsgi_app)
    self.init_authorizer()

  def __call__(self, environ, start_response):
    return self.wsgi_app(environ, start_response)

  def init_authorizer(self):
    if self.config:
      # custom init with JSON
      initAuthorizer(json.dumps(config))
    else:
      # use defaults with env
      initAuthorizer()

class Bundle(object):

class Label(object):

class Context(object):


def create_wsgi_middleware(other_wsgi):
  """
  Create a wrapper middleware for another WSGI response handler
  """

  def wsgi_auth_middleware(environ, start_response):
    token = get_token(environ)
    try:
      user = authorized(token)
    except error as e:
      status = '401 Unauthorized'
      start_response(status, '')
      return [ str(e) ]

    try:
      passed = checkGlobalRequirements(user, token)
      if not passed:
        status = '403 Forbidden'
        start_response(status,'')
        return ['not allowed to access this resource' ]
    except error as e:
      status = '403 Forbidden'
      start_response(status,'')
      return [ str(e) ]

    environ['remote_user'] = user
    return other_wsgi(environ, start_response)

  return wsgi_auth_middleware


def get_token(environ):
  """
  Utility method to retrieve the access token from request
  """
  request = Request(environ)
  token = ""

  header = request.headers.get('Authorization','')
  if header != '':
    # token in header
    prefix = 'Bearer '
    if header.startswith(prefix):
        token = header[len(prefix):]
  if token =="":
    # query param
    token = request.args.get('access_token', '')
  if token == '':
    request_body_size = request.content_length
    if request_body_size>0:
      # urlencoded form param
      # multipart/data form param
      token = request.form.get('access_token', '')

  return token


def get_username(environ):
  """
  Utility method to retrieve the authenticated username
  """
  if environ['remote_user']:
      return environ['remote_user']
  return ''

def acl(requirements):
  """
  Decorator to define ACL checks per route.

  Configure a route like this:

  @app.route('/resources')
  @acl({
  "groups": [ "group1", "group2" ],
  "roles": [ "role1" ],
  "access_policies": { "resource1": "action1", "resource2": "action2" }
  })
  def fetch_resources():
    ...

  Access policy check requires a remote authorizer to be deployed.
  Groups and roles are directly checked against token claims.
  If action is left empty, the resource is checked against any action
  """
  def wrapped_with_args(entrypoint):
    @functools.wraps(entrypoint)
    def wrapped_entrypoint(**kwargs):
      try:
        requirementsAsJSON = json.dumps(requirements)
        passed = allowed(get_username(request.environ), get_token(request.environ), requirementsAsJSON)
        if not passed:
          resp = make_response('not allowed to access this resource', 403)
          return resp
      except error as e:
        resp = make_response(str(e), 403)
        return resp
      return entrypoint(**kwargs)
    return wrapped_entrypoint
  return wrapped_with_args
