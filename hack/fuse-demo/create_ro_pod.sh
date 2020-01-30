#! /bin/zsh

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

proj_root_dir="$(dirname "$(dirname "$SCRIPT_DIR")")"

DATAMON_EXEC="$proj_root_dir"/cmd/datamon/run_datamon

repo_name=
label=
bundle_id=

while getopts r:l:b: opt; do
    case $opt in
        (r)
            repo_name="$OPTARG"
            ;;
        (l)
            label="$OPTARG"
            ;;
        (b)
            bundle_id="$OPTARG"
            ;;
        (\?)
            print Bad option, aborting.
            exit 1
            ;;
    esac
done
(( OPTIND > 1 )) && shift $(( OPTIND - 1 ))

if [ -z $repo_name ]; then
    repo_name='ransom-datamon-test-repo'
fi

if [ -z $label ]; then
    if [ -z $bundle_id ]; then
        label='testlabel'
    fi
fi

if [[ -z $GOOGLE_APPLICATION_CREDENTIALS ]]; then
	echo 'GOOGLE_APPLICATION_CREDENTIALS env variable not set' 1>&2
	exit 1
fi

if kubectl get secret google-application-credentials &> /dev/null; then
	kubectl delete secret google-application-credentials
fi

# https://cloud.google.com/kubernetes-engine/docs/tutorials/authenticating-to-cloud-platform#step_4_import_credentials_as_a_secret
kubectl create secret generic \
	google-application-credentials \
	--from-file=google-application-credentials.json=$GOOGLE_APPLICATION_CREDENTIALS

# PROJROOT="$(git rev-parse --show-toplevel)" \
    # GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD |sed 's@/@_@g')" \

echo "have repo $repo_name label $label bundle id $bundle_id"

if [ -z $label ]; then
    if [ -z $bundle_id ]; then
        echo 'no bundle specified' 1>&2
        exit 1
    fi
else
    if [ ! "$bundle_id" = '' ]; then
        echo 'bundle id and label are mutually exclusive' 1>&2
        exit 1
    fi
    bundle_id=$("$DATAMON_EXEC" label get --repo "$repo_name" --label "$label" | \
                    cut -d ',' -f 2 | tr -d ' ')
    if [ "$bundle_id" = '' ]; then
        echo "couldn't find bundle if for label $label" 1>&2
        exit 1
    fi
fi

RES_DEF="$proj_root_dir"/hack/k8s/gen/example-ro.yaml

SHELL_NAME="$(basename "$SHELL")" \
          REPO_NAME="$repo_name" \
          BUNDLE_ID="$bundle_id" \
	"$proj_root_dir"/hack/envexpand "$proj_root_dir"/hack/k8s/example-ro.template.yaml > "$RES_DEF"

if kubectl get deployment datamon-ro-demo &> /dev/null; then
	kubectl delete -f "$RES_DEF"
fi

kubectl create -f "$RES_DEF"
