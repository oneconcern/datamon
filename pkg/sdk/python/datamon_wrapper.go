#define Py_AUTHORIZER_API
#include <Python.h>
#include <authorizer.h>

static PyObject *AuthorizerError;

// py_newAuthorizer() is the python wrapper for sdkAuthNewAuthorizer()
static PyObject *
py_newAuthorizer(PyObject *self, PyObject *args) {
    char *json;
    char *err;
    int r;

    json=NULL;
    if (!PyArg_ParseTuple(args, "|s",&json)) {
        PyErr_SetString(AuthorizerError, "new may take at most 1 parameter. Usage: newAuthorizer([json])");
        return NULL;
    }
    r = sdkAuthNewAuthorizer(json,&err);
    if (r < 0) {
        PyErr_SetString(AuthorizerError, err);
        free(err);
        return NULL;
    }
    return Py_BuildValue("i",r);
}

// py_allowed() is the python wrapper for sdkAuthAllowed()
static PyObject *
py_allowed(PyObject *self, PyObject *args) {
    char *user;
    char *token;
    char *requirementsAsJSON;
    char *err;
    int r;

    if (!PyArg_ParseTuple(args, "sss",&user,&token,&requirementsAsJSON)) {
        PyErr_SetString(AuthorizerError, "allowed requires 3 parameters. Usage: allowed(user,token,requirementsAsJSON)");
        return NULL;
    }
    r = sdkAuthAllowed(user,token,requirementsAsJSON,&err);
    if (r < 0) {
        PyErr_SetString(AuthorizerError, err);
        free(err);
        return NULL;
    }
    return Py_BuildValue("i",r);
}

// py_authorized() is the python wrapper for sdkAuthAuthorized()
static PyObject *
py_authorized(PyObject *self, PyObject *args) {
    char *token;
    char *err;
    char *user ;
    PyObject *ret;

    if (!PyArg_ParseTuple(args, "s",&token)) {
        PyErr_SetString(AuthorizerError, "authorize requires 1 parameter. Usage: authorize(token)");
        return NULL;
    }

    user = sdkAuthAuthorized(token, &err);
    if (user == NULL) {
        if (err == NULL) {
        }
        PyErr_SetString(AuthorizerError,err);
        free(err);
        return NULL;
    }
    ret = Py_BuildValue("s",user);
    free(user);
    return ret;
}

// py_checkGlobalRequirements() is the python wrapper for sdkAuthCheckGlobalRequirements()
static PyObject *
py_checkGlobalRequirements(PyObject *self, PyObject *args) {
    char *user;
    char *token;
    char *err;
    int r;

    if (!PyArg_ParseTuple(args, "ss",&user,&token)) {
        PyErr_SetString(AuthorizerError, "checkGlobalRequirements requires 2 parameters. Usage: checkGlobalRequirements(user,token)");
        return NULL;
    }
    r = sdkAuthCheckGlobalRequirements(user,token,&err);
    if (r < 0) {
        PyErr_SetString(AuthorizerError, err);
        free(err);
        return NULL;
    }
    return Py_BuildValue("i",r);
}

static PyMethodDef AuthorizerMethods[] = {
    {"initAuthorizer", (PyCFunction)py_newAuthorizer, METH_VARARGS, "Creates and initializes a new authorizer instance"},
    {"allowed", (PyCFunction)py_allowed, METH_VARARGS, "Verifies an ACL requirement"},
    {"authorized", (PyCFunction)py_authorized, METH_VARARGS, "Verifies an authentication token"},
    {"checkGlobalRequirements", (PyCFunction)py_checkGlobalRequirements, METH_VARARGS, "Verifies ACLs configured for this authorizer"},
    {NULL, NULL, 0, NULL}
};

#ifdef Py_InitModule
/* Python 2.7 module initialization */
PyMODINIT_FUNC initauthorizer(void) {
    PyObject *m;

    sdkAuthInitAuthorizer();
    m = Py_InitModule("authorizer",AuthorizerMethods);
    if (m == NULL) {
        return;
    }

    AuthorizerError = PyErr_NewException("authorizer.error", NULL, NULL);
    Py_INCREF(AuthorizerError);
    PyModule_AddObject(m, "error", AuthorizerError);
}
#else
/* Python 3.5+ module initialization */
static struct PyModuleDef authModule =
{
    PyModuleDef_HEAD_INIT,
    "authorizer",
    "authorizer is a module to authenticate and authorize API endpoints",
    -1,
    AuthorizerMethods
};

PyMODINIT_FUNC PyInit_authorizer(void)
{
    PyObject *m;
    sdkAuthInitAuthorizer();

    m = PyModule_Create(&authModule);
    if (m == NULL) {
        return NULL;
    }

    AuthorizerError = PyErr_NewException("authorizer.error", NULL, NULL);
    Py_INCREF(AuthorizerError);
    PyModule_AddObject(m, "error", AuthorizerError);

    return m;
}
#endif
