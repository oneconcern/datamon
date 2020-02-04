#define Py_AUTHORIZER_API
#include <Python.h>
#include <datamon.h>

static PyObject *DatamonError;

// py_listRepos() is the python wrapper for listRepos()
static PyObject *
py_listRepos(PyObject *self, PyObject *args) {
    char *config;
    char *result;
    char *err;
    int r;
    PyObject *ret;

    config=NULL;
    // TODO(fred): as a target I want the config parameter to be opt-in ("|s")
    if (!PyArg_ParseTuple(args, "s",&config)) {
        PyErr_SetString(DatamonError, "listRepos requires 1 parameter: config");
        return NULL;
    }
    r = listRepos(config,&result, &err);
    if (r < 0) {
        PyErr_SetString(DatamonError, err);
        free(err);
        return NULL;
    }
    ret = Py_BuildValue("s",result);
    free(result);
    return ret;
}

// py_listBundles() is the python wrapper for listBundles()
static PyObject *
py_listBundles(PyObject *self, PyObject *args) {
    char *config;
    char *repo;
    char *result;
    char *err;
    int r;
    PyObject *ret;

    config=NULL;
    if (!PyArg_ParseTuple(args, "ss",&config,&repo)) {
        PyErr_SetString(DatamonError, "listBundles requires 2 parameters: config, repo");
        return NULL;
    }
    r = listBundles(config,repo,&result, &err);
    if (r < 0) {
        PyErr_SetString(DatamonError, err);
        free(err);
        return NULL;
    }
    ret = Py_BuildValue("s",result);
    free(result);
    return ret;
}

static PyMethodDef DatamonMethods[] = {
    {"listRepos", (PyCFunction)py_listRepos, METH_VARARGS, "List all datamon repos"},
    {"listBundles", (PyCFunction)py_listBundles, METH_VARARGS, "List all bundles in a repo"},
    {NULL, NULL, 0, NULL}
};

#ifdef Py_InitModule
/* Python 2.7 module initialization */
PyMODINIT_FUNC initdatamon(void) {
    PyObject *m;
    m = Py_InitModule("datamon",DatamonMethods);
    if (m == NULL) {
        return;
    }
    DatamonError = PyErr_NewException("datamon.error", NULL, NULL);
    Py_INCREF(DatamonError);
    PyModule_AddObject(m, "error", DatamonError);
}
#else
/* Python 3.5+ module initialization */
static struct PyModuleDef datamonModule =
{
    PyModuleDef_HEAD_INIT,
    "datamon",
    "datamon is a module to manage data at scale",
    -1,
    DatamonMethods
};

PyMODINIT_FUNC PyInit_datamon(void)
{
    PyObject *m;
    m = PyModule_Create(&datamonModule);
    if (m == NULL) {
        return NULL;
    }
    DatamonError = PyErr_NewException("datamon.error", NULL, NULL);
    Py_INCREF(DatamonError);
    PyModule_AddObject(m, "error", DatamonError);
    return m;
}
#endif
