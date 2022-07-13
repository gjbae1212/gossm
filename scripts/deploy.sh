#!/bin/bash
set -e -o pipefail
trap '[ "$?" -eq 0 ] || echo "Error Line:<$LINENO> Error Function:<${FUNCNAME}>"' EXIT
cd `dirname $0` && cd ..
CURRENT=`pwd`

function test
{
    go test -v $(go list ./... | grep -v vendor) --count 1 -race -coverprofile=$CURRENT/coverage.txt -covermode=atomic
}

function test_with_circleci
{
    export CIRCLECI="true"
    go test -v $(go list ./... | grep -v vendor) --count 1 -race -coverprofile=$CURRENT/coverage.txt -covermode=atomic
}

function release
{
  sudo rm -rf $CURRENT/dist
  sudo rm -rf $CURRENT/gopath
  export GOPATH=$CURRENT/gopath

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

function release_test
{
  sudo rm -rf $CURRENT/dist
  sudo rm -rf $CURRENT/gopath
  export GOPATH=$CURRENT/gopath
  goreleaser release --snapshot --rm-dist
}

CMD=$1
shift
$CMD $*
