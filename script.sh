#!/bin/bash
set -e -o pipefail
trap '[ "$?" -eq 0 ] || echo "Error Line:<$LINENO> Error Function:<${FUNCNAME}>"' EXIT
cd `dirname $0`
CURRENT=`pwd`

function test
{
    go test -v $(go list ./... | grep -v vendor) --count 1 -race -coverprofile=$CURRENT/coverage.txt -covermode=atomic
}

function build
{
   packr2 && go build
}

function release
{

  tag=$1
  if [ -z "$tag" ]
  then
     echo "not found tag name"
     exit 1
  fi

  git tag -a $tag -m "Add $tag"
  git push origin $tag

  goreleaser release --rm-dist
}

CMD=$1
shift
$CMD $*
