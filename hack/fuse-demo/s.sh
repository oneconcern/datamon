#! /bin/bash
 kubectl -n datamon-ci create secret generic google-application-credentials --from-file ~/.config/gcloud/application_default_credentials.json
